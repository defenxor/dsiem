package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/time/rate"

	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/dsiem/pkg/siem"
	log "dsiem/internal/shared/pkg/logger"
	"dsiem/internal/shared/pkg/str"

	"github.com/remeh/sizedwaitgroup"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	progName = "dtester"
)

var version string
var buildTime string

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().StringP("address", "a", "127.0.0.1", "Dsiem IP address to send events to")
	testCmd.Flags().IntP("port", "p", 8080, "Dsiem TCP port")
	testCmd.Flags().IntP("max", "m", 1000, "Maximum number of events to send per rule")
	testCmd.Flags().StringP("homenet", "i", "192.168.0.1", "IP address to use to represent HOME_NET. This IP must already be defined in dsiem assets configuration")
	testCmd.Flags().StringP("file", "f", "directives_*.json", "file glob pattern to load directives from")
	testCmd.Flags().IntP("rps", "r", 500, "number of HTTP post request per second")
	testCmd.Flags().IntP("concurrency", "c", 50, "number of concurrent HTTP post to submit")
	testCmd.Flags().BoolP("verbose", "v", false, "print sent events to console")
	viper.BindPFlag("address", testCmd.Flags().Lookup("address"))
	viper.BindPFlag("port", testCmd.Flags().Lookup("port"))
	viper.BindPFlag("homenet", testCmd.Flags().Lookup("homenet"))
	viper.BindPFlag("file", testCmd.Flags().Lookup("file"))
	viper.BindPFlag("max", testCmd.Flags().Lookup("max"))
	viper.BindPFlag("rps", testCmd.Flags().Lookup("rps"))
	viper.BindPFlag("concurrency", testCmd.Flags().Lookup("concurrency"))
	viper.BindPFlag("verbose", testCmd.Flags().Lookup("verbose"))
}

func initConfig() {
	viper.SetEnvPrefix(progName)
	viper.AutomaticEnv()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		exit("Error returned from command", err)
	}
}

func exit(msg string, err error) {
	fmt.Println("Exiting: " + msg + ": " + err.Error())
	os.Exit(1)
}

var rootCmd = &cobra.Command{
	Use:   "dtester",
	Short: "Directive rules tester for Dsiem",
	Long:  `Dtester test directive rules by sending a matching event to dsiem`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and build information",
	Long:  `Print the version and build date information`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version, buildTime)
	},
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Start sending events",
	Long: `
Send events to dsiem at defined address and port`,
	Run: func(cmd *cobra.Command, args []string) {

		log.Setup(true)

		addr := viper.GetString("address")
		port := viper.GetInt("port")
		file := viper.GetString("file")

		dirs, _, err := siem.LoadDirectivesFromFile("", file)
		if err != nil {
			exit("Cannot initialize directives", err)
		}
		sender(&dirs, addr, port)
	},
}

func sender(d *siem.Directives, addr string, port int) {
	max := viper.GetInt("max")
	// conc := viper.GetInt("concurrency")
	rps := viper.GetInt("rps")
	conc := viper.GetInt("concurrency")
	verbose := viper.GetBool("verbose")
	// conc := rps / 50
	//conc = 1
	//time.Sleep(3 * time.Second)
	//fmt.Println("using conc: ", conc)
	//time.Sleep(5 * time.Second)
	keepAliveTimeout := 600 * time.Second
	timeout := 5 * time.Second

	defaultTransport := &http.Transport{
		Dial:                (&net.Dialer{KeepAlive: keepAliveTimeout}).Dial,
		MaxIdleConns:        conc + 100,
		MaxIdleConnsPerHost: conc + 100,
	}
	c := &http.Client{
		Transport: defaultTransport,
		Timeout:   timeout,
	}
	swg := sizedwaitgroup.New(conc)
	fn := rateLimit(rps, rps, timeout, sendHTTPSingleConn)

	for _, v := range d.Dirs {
		var prevPortTo int
		var prevPortFrom int
		var prevFrom string
		var prevTo string
		for _, j := range v.Rules {
			amt := j.Occurrence
			if amt > max {
				amt = max
			}
			e := event.NormalizedEvent{}
			e.Sensor = progName
			e.SrcIP = genIP(j.From, prevFrom)
			e.DstIP = genIP(j.To, prevTo)
			e.SrcPort = genPort(j.PortFrom, prevPortFrom, false)
			e.DstPort = genPort(j.PortTo, prevPortTo, true)
			e.Protocol = genProto(j.Protocol)
			e.PluginID = j.PluginID
			e.PluginSID = pickOneFromIntSlice(j.PluginSID)
			e.Product = pickOneFromStrSlice(j.Product)
			e.Category = j.Category
			e.SubCategory = pickOneFromStrSlice(j.SubCategory)

			prevPortTo = e.DstPort
			prevPortFrom = e.SrcPort
			prevFrom = e.SrcIP
			prevTo = e.DstIP

			for i := 0; i < amt; i++ {
				swg.Add()
				go func(st int, iter int) {
					defer swg.Done()
					for {
						for {
							//	err := fn(&e, c, st, iter, verbose)
							err := fn(&e, c, j.Stage, i, verbose)
							if err != nil {
								log.Info(log.M{Msg: "Received error: " + err.Error() + ". Retrying in 3 second."})
								time.Sleep(3 * time.Second)
								continue
							}
							break
						}
					}
				}(j.Stage, i)
			}
		}
	}
	swg.Wait()
}

type eventPoster func(e *event.NormalizedEvent, c *http.Client, stage int, iter int, verbose bool) error

func rateLimit(rps, burst int, wait time.Duration, h eventPoster) eventPoster {
	l := rate.NewLimiter(rate.Limit(rps), burst)

	return func(e *event.NormalizedEvent, c *http.Client, stage int, iter int, verbose bool) error {
		// create a new context from the request with the wait timeout
		ctx, cancel := context.WithTimeout(context.Background(), wait)
		defer cancel() // always cancel the context!

		// Wait errors out if the request cannot be processed within
		// the deadline. This is preemptive, instead of waiting the
		// entire duration.
		if err := l.Wait(ctx); err != nil {
			return err
		}
		return h(e, c, stage, iter, verbose)
	}
}

func sendHTTPSingleConn(e *event.NormalizedEvent, c *http.Client, stage int, iter int, verbose bool) error {
	e.EventID = genUUID()
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	evt := string(b)
	url := "http://" + viper.GetString("address") + ":" + strconv.Itoa(viper.GetInt("port")) + "/events/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	if err != nil {
		//log.Warn(log.M{Msg: "Cannot create new HTTP request, " + err.Error()})
		return err
	}
	resp, err := c.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		//log.Warn(log.M{Msg: "Failed to send event to dsiem, consider lowering EPS or concurrency setting: " + err.Error()})
		return err
	}

	_, _ = io.Copy(ioutil.Discard, resp.Body) // read the body to avoid mem leak?

	if verbose {
		fmt.Println("Sent event for stage:", stage, "order #:", iter)
		fmt.Println(evt)
	}
	return nil
}

func randInt(min int, max int) int {
	m := max - min
	if max < min || m == 0 {
		return 0
	}
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Intn(m) + min
}

func genPort(portList string, prev int, useLowPort bool) int {

	if _, ok := str.RefToDigit(portList); ok {
		return prev
	}

	if portList == "ANY" {
		if useLowPort {
			return randInt(20, 1024)
		} else {
			return randInt(4000, 65535)
		}
	}
	s := pickOneFromCsv(portList)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func genIP(ruleAddr string, prev string) string {
	if ruleAddr == "HOME_NET" {
		return viper.GetString("homenet")
	}

	if _, ok := str.RefToDigit(ruleAddr); ok {
		return prev
	}

	// testing, always return prev
	if prev != "" {
		return prev
	}

	var octet [4]int
	for octet[0] == 0 || octet[0] == 10 || octet[0] == 127 || octet[0] == 192 || octet[0] == 172 {
		octet[0] = randInt(1, 254)
	}
	for i := 1; i <= 3; i++ {
		octet[i] = randInt(1, 254)
	}
	sIP := strconv.Itoa(octet[0]) + "." + strconv.Itoa(octet[1]) +
		"." + strconv.Itoa(octet[2]) + "." + strconv.Itoa(octet[3])
	return sIP
}

func genProto(proto string) string {
	if proto == "ANY" {
		return pickOneFromCsv("TCP,UDP")
	}
	return proto
}

func pickOneFromCsv(s string) string {
	sSlice := str.CsvToSlice(s)
	l := len(sSlice)
	return sSlice[randInt(1, l)]
}

func pickOneFromStrSlice(s []string) string {
	l := len(s)
	if l == 0 {
		return ""
	}
	return s[randInt(1, l)]
}

func pickOneFromIntSlice(n []int) int {
	l := len(n)
	if l == 0 {
		return 0
	}
	return n[randInt(1, l)]
}

func genUUID() string {
	u, err := uuid.NewV4()
	if err != nil {
		return "static-id-doesnt-really-matter"
	}
	return u.String()
}
