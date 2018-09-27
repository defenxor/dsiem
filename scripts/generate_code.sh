#!/bin/bash

gojay -s internal/dsiem/pkg/event/event.go -t NormalizedEvent > internal/dsiem/pkg/event/event_gen.go
gojay -s internal/dsiem/pkg/siem/backlog.go -t siemAlarmEvents > internal/dsiem/pkg/siem/backlog_gen.go


