package main

import (
	"flag"
	"net/http"
	"os"
	"render-deploy-webhook-sender/service/rendercom"
	"strings"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/log"

	"github.com/peterbourgon/ff/v3"
)

func main() {
	fs := flag.NewFlagSet("webhook-receiver", flag.ExitOnError)
	var (
		environment  = fs.String("environment", "develop", "the environment we are running in")
		apiToken     = fs.String("api-token", "changeme", "the secret token for the api")
		webhookURL   = fs.String("webhook-url", "https://example.com/123/hook", "the url of the webhook we should hit if there was a new deploy")
		deployWindow = fs.Int("deploy-window", 10, "the deploy window in minutes. if within this timeframe there was a successful deploy we call the webhook")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
	)

	l := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	switch strings.ToLower(*environment) {
	case "development":
		l = level.NewFilter(l, level.AllowInfo())
	case "prod":
		l = level.NewFilter(l, level.AllowError())
	}
	l = log.With(l, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	c := http.DefaultClient
	rs := rendercom.NewRenderService(c, *apiToken)
	s, err := rs.GetServices()
	if err != nil {
		level.Error(l).Log("err", err)
		return
	}
	for _, service := range s {
		if service.Service.Suspended != "not_suspended" {
			continue
		}
		level.Info(l).Log("msg", "checking service", "name", service.Service.Name)
		d, err := rs.GetDeploys(service.Service.Id, "live")
		if err != nil {
			level.Error(l).Log("err", err)
			return
		}
		// There should only be one "live" deploy at any given time
		if len(d) != 1 {
			level.Error(l).Log("msg", "there are more than one deploy live, this should not happen")
			return
		}
		finishedAt := d[0].Deploy.FinishedAt
		finishedAt = time.Now().Add(time.Duration(-2) * time.Minute)
		if finishedAt.After(time.Now().Add(time.Duration(-*deployWindow) * time.Minute)) {
			level.Info(l).Log("msg", "found deploy within deploy window", "deploy_ts", finishedAt, "now_ts", time.Now())
			resp, err := c.Post(*webhookURL, "application/json", nil)
			if err != nil {
				level.Error(l).Log("err", err)
				return
			}
			if resp.StatusCode == http.StatusAccepted {
				level.Info(l).Log("msg", "webhook hit successfully", "status_code", resp.StatusCode)
				return
			} else {
				level.Error(l).Log("msg", "unexpected status code", "status_code", resp.StatusCode)
				return
			}
		}
	}
}
