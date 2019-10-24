// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package apm

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"go.elastic.co/apm/internal/configutil"
	"go.elastic.co/apm/internal/wildcard"
	"go.elastic.co/apm/model"
)

const (
	envMetricsInterval       = "ELASTIC_APM_METRICS_INTERVAL"
	envMaxSpans              = "ELASTIC_APM_TRANSACTION_MAX_SPANS"
	envTransactionSampleRate = "ELASTIC_APM_TRANSACTION_SAMPLE_RATE"
	envSanitizeFieldNames    = "ELASTIC_APM_SANITIZE_FIELD_NAMES"
	envCaptureHeaders        = "ELASTIC_APM_CAPTURE_HEADERS"
	envCaptureBody           = "ELASTIC_APM_CAPTURE_BODY"
	envServiceName           = "ELASTIC_APM_SERVICE_NAME"
	envServiceVersion        = "ELASTIC_APM_SERVICE_VERSION"
	envEnvironment           = "ELASTIC_APM_ENVIRONMENT"
	envSpanFramesMinDuration = "ELASTIC_APM_SPAN_FRAMES_MIN_DURATION"
	envActive                = "ELASTIC_APM_ACTIVE"
	envAPIRequestSize        = "ELASTIC_APM_API_REQUEST_SIZE"
	envAPIRequestTime        = "ELASTIC_APM_API_REQUEST_TIME"
	envAPIBufferSize         = "ELASTIC_APM_API_BUFFER_SIZE"
	envMetricsBufferSize     = "ELASTIC_APM_METRICS_BUFFER_SIZE"
	envDisableMetrics        = "ELASTIC_APM_DISABLE_METRICS"
	envGlobalLabels          = "ELASTIC_APM_GLOBAL_LABELS"
	envStackTraceLimit       = "ELASTIC_APM_STACK_TRACE_LIMIT"
	envCentralConfig         = "ELASTIC_APM_CENTRAL_CONFIG"
	envBreakdownMetrics      = "ELASTIC_APM_BREAKDOWN_METRICS"

	defaultAPIRequestSize        = 750 * configutil.KByte
	defaultAPIRequestTime        = 10 * time.Second
	defaultAPIBufferSize         = 1 * configutil.MByte
	defaultMetricsBufferSize     = 750 * configutil.KByte
	defaultMetricsInterval       = 30 * time.Second
	defaultMaxSpans              = 500
	defaultCaptureHeaders        = true
	defaultCaptureBody           = CaptureBodyOff
	defaultSpanFramesMinDuration = 5 * time.Millisecond
	defaultStackTraceLimit       = 50

	minAPIBufferSize     = 10 * configutil.KByte
	maxAPIBufferSize     = 100 * configutil.MByte
	minAPIRequestSize    = 1 * configutil.KByte
	maxAPIRequestSize    = 5 * configutil.MByte
	minMetricsBufferSize = 10 * configutil.KByte
	maxMetricsBufferSize = 100 * configutil.MByte
)

var (
	defaultSanitizedFieldNames = configutil.ParseWildcardPatterns(strings.Join([]string{
		"password",
		"passwd",
		"pwd",
		"secret",
		"*key",
		"*token*",
		"*session*",
		"*credit*",
		"*card*",
		"authorization",
		"set-cookie",
	}, ","))

	globalLabels = func() model.StringMap {
		var labels model.StringMap
		for _, kv := range configutil.ParseListEnv(envGlobalLabels, ",", nil) {
			i := strings.IndexRune(kv, '=')
			if i > 0 {
				k, v := strings.TrimSpace(kv[:i]), strings.TrimSpace(kv[i+1:])
				labels = append(labels, model.StringMapItem{
					Key:   cleanTagKey(k),
					Value: truncateString(v),
				})
			}
		}
		return labels
	}()
)

func initialRequestDuration() (time.Duration, error) {
	return configutil.ParseDurationEnv(envAPIRequestTime, defaultAPIRequestTime)
}

func initialMetricsInterval() (time.Duration, error) {
	return configutil.ParseDurationEnv(envMetricsInterval, defaultMetricsInterval)
}

func initialMetricsBufferSize() (int, error) {
	size, err := configutil.ParseSizeEnv(envMetricsBufferSize, defaultMetricsBufferSize)
	if err != nil {
		return 0, err
	}
	if size < minMetricsBufferSize || size > maxMetricsBufferSize {
		return 0, errors.Errorf(
			"%s must be at least %s and less than %s, got %s",
			envMetricsBufferSize, minMetricsBufferSize, maxMetricsBufferSize, size,
		)
	}
	return int(size), nil
}

func initialAPIBufferSize() (int, error) {
	size, err := configutil.ParseSizeEnv(envAPIBufferSize, defaultAPIBufferSize)
	if err != nil {
		return 0, err
	}
	if size < minAPIBufferSize || size > maxAPIBufferSize {
		return 0, errors.Errorf(
			"%s must be at least %s and less than %s, got %s",
			envAPIBufferSize, minAPIBufferSize, maxAPIBufferSize, size,
		)
	}
	return int(size), nil
}

func initialAPIRequestSize() (int, error) {
	size, err := configutil.ParseSizeEnv(envAPIRequestSize, defaultAPIRequestSize)
	if err != nil {
		return 0, err
	}
	if size < minAPIRequestSize || size > maxAPIRequestSize {
		return 0, errors.Errorf(
			"%s must be at least %s and less than %s, got %s",
			envAPIRequestSize, minAPIRequestSize, maxAPIRequestSize, size,
		)
	}
	return int(size), nil
}

func initialMaxSpans() (int, error) {
	value := os.Getenv(envMaxSpans)
	if value == "" {
		return defaultMaxSpans, nil
	}
	max, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse %s", envMaxSpans)
	}
	return max, nil
}

// initialSampler returns a nil Sampler if all transactions should be sampled.
func initialSampler() (Sampler, error) {
	value := os.Getenv(envTransactionSampleRate)
	return parseSampleRate(envTransactionSampleRate, value)
}

// parseSampleRate parses a numeric sampling rate in the range [0,1.0], returning a Sampler.
func parseSampleRate(name, value string) (Sampler, error) {
	if value == "" {
		value = "1"
	}
	ratio, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", name)
	}
	if ratio < 0.0 || ratio > 1.0 {
		return nil, errors.Errorf(
			"invalid value for %s: %s (out of range [0,1.0])",
			name, value,
		)
	}
	return NewRatioSampler(ratio), nil
}

func initialSanitizedFieldNames() wildcard.Matchers {
	return configutil.ParseWildcardPatternsEnv(envSanitizeFieldNames, defaultSanitizedFieldNames)
}

func initialCaptureHeaders() (bool, error) {
	return configutil.ParseBoolEnv(envCaptureHeaders, defaultCaptureHeaders)
}

func initialCaptureBody() (CaptureBodyMode, error) {
	value := os.Getenv(envCaptureBody)
	if value == "" {
		return defaultCaptureBody, nil
	}
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "all":
		return CaptureBodyAll, nil
	case "errors":
		return CaptureBodyErrors, nil
	case "transactions":
		return CaptureBodyTransactions, nil
	case "off":
		return CaptureBodyOff, nil
	}
	return -1, errors.Errorf("invalid %s value %q", envCaptureBody, value)
}

func initialService() (name, version, environment string) {
	name = os.Getenv(envServiceName)
	version = os.Getenv(envServiceVersion)
	environment = os.Getenv(envEnvironment)
	if name == "" {
		name = filepath.Base(os.Args[0])
		if runtime.GOOS == "windows" {
			name = strings.TrimSuffix(name, filepath.Ext(name))
		}
	}
	name = sanitizeServiceName(name)
	return name, version, environment
}

func initialSpanFramesMinDuration() (time.Duration, error) {
	return configutil.ParseDurationEnv(envSpanFramesMinDuration, defaultSpanFramesMinDuration)
}

func initialActive() (bool, error) {
	return configutil.ParseBoolEnv(envActive, true)
}

func initialDisabledMetrics() wildcard.Matchers {
	return configutil.ParseWildcardPatternsEnv(envDisableMetrics, nil)
}

func initialStackTraceLimit() (int, error) {
	value := os.Getenv(envStackTraceLimit)
	if value == "" {
		return defaultStackTraceLimit, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse %s", envStackTraceLimit)
	}
	return limit, nil
}

func initialCentralConfigEnabled() (bool, error) {
	return configutil.ParseBoolEnv(envCentralConfig, true)
}

func initialBreakdownMetricsEnabled() (bool, error) {
	return configutil.ParseBoolEnv(envBreakdownMetrics, true)
}
