package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/remeh/sizedwaitgroup"

	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/dsiem/pkg/siem"
	log "dsiem/internal/shared/pkg/logger"
	"dsiem/internal/shared/pkg/str"

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
	testCmd.Flags().IntP("concurrency", "c", 500, "number of HTTP post to send concurrently")
	testCmd.Flags().BoolP("verbose", "v", false, "print sent events to console")
	viper.BindPFlag("address", testCmd.Flags().Lookup("address"))
	viper.BindPFlag("concurrency", testCmd.Flags().Lookup("concurrency"))
	viper.BindPFlag("port", testCmd.Flags().Lookup("port"))
	viper.BindPFlag("homenet", testCmd.Flags().Lookup("homenet"))
	viper.BindPFlag("file", testCmd.Flags().Lookup("file"))
	viper.BindPFlag("max", testCmd.Flags().Lookup("max"))
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
	if viper.GetBool("debug") {
		fmt.Println(msg)
		panic(err)
	} else {
		fmt.Println("Exiting: " + msg + ": " + err.Error())
		os.Exit(1)
	}
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

		log.Setup(false)

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
	conc := viper.GetInt("concurrency")
	verbose := viper.GetBool("verbose")
	c := http.Client{Timeout: time.Second * 5}
	swg := sizedwaitgroup.New(conc)

	for _, v := range d.Dirs {
		var prevPortTo string
		var prevPortFrom string
		var prevFrom string
		var prevTo string
		for _, j := range v.Rules {
			amt := j.Occurrence
			if amt > max {
				amt = max
			}
			e := event.NormalizedEvent{}
			e.Sensor = progName
			e.Timestamp = time.Now().UTC().Format(time.RFC3339)
			e.EventID = genUUID()
			e.SrcIP = genIP(j.From)
			e.DstIP = genIP(j.To)
			e.SrcPort = genPort(j.PortFrom, false)
			e.DstPort = genPort(j.PortTo, true)
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
					sendHTTPSingleConn(&e, &c, st, iter, verbose)
				}(j.Stage, i)
			}
		}
	}
	swg.Wait()
}

func sendHTTPSingleConn(e *event.NormalizedEvent, c *http.Client, stage int, iter int, verbose bool) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	evt := string(b)
	url := "http://" + viper.GetString("address") + ":" + strconv.Itoa(viper.GetInt("port")) + "/events/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	req.Header.Set("Connection", "close")
	if err != nil {
		log.Warn(log.M{Msg: "Cannot create new HTTP request, " + err.Error()})
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		log.Warn(log.M{Msg: "Failed to send event to dsiem, consider lowering concurrency setting: " + err.Error()})
		return err
	}
	resp.Body.Close()

	if verbose {
		fmt.Println("stage:", stage, "iter:", iter)
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

func genPort(portList string, useLowPort bool) int {
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

func genIP(ruleAddr string) string {
	if ruleAddr == "HOME_NET" {
		return viper.GetString("homenet")
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
