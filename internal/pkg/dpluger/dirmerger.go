package dpluger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
)

type Commander interface {
	PromptBool(string, bool) bool
	Log(string)
}

type FileReader interface {
	Read(string) ([]byte, error)
}

type MergeConfig struct {
	Host       string
	SourceJSON string
	TargetJSON string
}

type mergeOption struct {
	transport  http.RoundTripper
	fileReader FileReader
}

type MergeOptionFunc func(*mergeOption)

func WithCustomTransport(tr http.RoundTripper) MergeOptionFunc {
	return func(o *mergeOption) {
		o.transport = tr
	}
}

func WithCustomFileReader(fr FileReader) MergeOptionFunc {
	return func(o *mergeOption) {
		o.fileReader = fr
	}
}

func Merge(cmd Commander, cfg MergeConfig, options ...MergeOptionFunc) error {
	opt := &mergeOption{}

	for _, option := range options {
		option(opt)
	}

	if opt.fileReader == nil {
		opt.fileReader = &defaultFileReader{}
	}

	httpClient := http.Client{}
	if opt.transport != nil {
		httpClient.Transport = opt.transport
	}

	if !strings.HasPrefix(cfg.Host, "http://") && !strings.HasPrefix(cfg.Host, "https://") {
		cfg.Host = fmt.Sprintf("http://%s", cfg.Host)
	}

	if !strings.HasSuffix(cfg.SourceJSON, ".json") {
		cfg.SourceJSON = fmt.Sprintf("%s.json", cfg.SourceJSON)
	}

	jsonURL := fmt.Sprintf("%s/config/%s", cfg.Host, cfg.SourceJSON)
	res, err := httpClient.Get(jsonURL)
	if err != nil {
		return fmt.Errorf("can not get existing file '%s', %s", cfg.SourceJSON, err.Error())
	}

	if res.StatusCode == http.StatusNotFound {
		res.Body.Close()
		return fmt.Errorf("can not find source JSON '%s'", cfg.SourceJSON)
	}

	if res.StatusCode == http.StatusForbidden {
		res.Body.Close()
		return fmt.Errorf("can not access source JSON '%s', access denied", cfg.SourceJSON)
	}

	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return fmt.Errorf("can not get source JSON '%s', %d", cfg.SourceJSON, res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		res.Body.Close()
		return fmt.Errorf("can not read source JSON file '%s', %s", cfg.SourceJSON, err.Error())
	}

	res.Body.Close()

	var dir siem.Directives
	if err := json.Unmarshal(b, &dir); err != nil {
		return fmt.Errorf("can not parse source JSON '%s', %s", cfg.SourceJSON, err.Error())
	}

	b, err = opt.fileReader.Read(cfg.TargetJSON)
	if err != nil {
		return fmt.Errorf("can not read target JSON '%s', %s", cfg.TargetJSON, err.Error())
	}

	var targetDir siem.Directives
	if err := json.Unmarshal(b, &targetDir); err != nil {
		return fmt.Errorf("can not parse target JSON '%s', %s", cfg.TargetJSON, err.Error())
	}

	resultDir := mergeDirectives(cmd, dir, targetDir)

	b, err = json.MarshalIndent(resultDir, "", "  ")
	if err != nil {
		return fmt.Errorf("can not parse result JSON for '%s', %s", cfg.SourceJSON, err.Error())
	}

	// push the directive back to the frontend
	res, err = httpClient.Post(jsonURL, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("can not apply merged directive, %s", err.Error())
	}

	defer res.Body.Close()

	// ensure connection reuse
	io.Copy(io.Discard, res.Body)

	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("can not apply merged directive, original directive not found")
	}

	if res.StatusCode == http.StatusForbidden {
		return fmt.Errorf("can not apply merged directive, access denied")
	}

	if res.StatusCode <= 200 || res.StatusCode > 299 {
		return fmt.Errorf("can not apply merged directive (%d)", res.StatusCode)
	}

	cmd.Log(fmt.Sprintf("file '%s' merged", cfg.SourceJSON))

	return nil
}

func mergeDirectives(cmd Commander, dir1, dir2 siem.Directives) siem.Directives {
	// map of directive.ID to its index in dir1.Dirs
	indexes := make(map[int]int)

	for index, directive := range dir1.Dirs {
		indexes[directive.ID] = index
	}

	newDirectives := make([]siem.Directive, 0)

	for _, directive := range dir2.Dirs {
		origIndex, ok := indexes[directive.ID]
		if !ok {
			newDirectives = append(newDirectives, directive)
			continue
		}

		same := compareDirective(dir1.Dirs[origIndex], directive)
		if same {
			continue
		}

		ok = cmd.PromptBool(fmt.Sprintf("directive #%d: '%s' is not equal to the existing directive, replace?", origIndex+1, dir1.Dirs[origIndex].Name), false)
		if ok {
			dir1.Dirs[origIndex] = directive
		} else {
			cmd.Log(fmt.Sprintf("change in directive #%d: '%s' omitted", origIndex+1, dir1.Dirs[origIndex].Name))
		}
	}

	return siem.Directives{
		Dirs: append(dir1.Dirs, newDirectives...),
	}
}

// compareDirective perform deep comparison between the two siem.Directive(s) and return true if they are equal.
func compareDirective(dir1, dir2 siem.Directive) bool {
	if err := directiveEqual(dir1, dir2); err != nil {
		return false
	}

	return true
}

type defaultFileReader struct {
}

func (f *defaultFileReader) Read(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}
