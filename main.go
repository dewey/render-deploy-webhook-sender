package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"render-deploy-webhook-sender/service/rendercom"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/robfig/cron/v3"

	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/log"

	"github.com/peterbourgon/ff/v3"
)

func main() {
	fs := flag.NewFlagSet("webhook-receiver", flag.ExitOnError)
	var (
		environment  = fs.String("environment", "develop", "the environment we are running in")
		port         = fs.String("port", "8080", "the port render-deploy-webhook-sender is running on")
		apiToken     = fs.String("api-token", "", "the secret token for the api")
		serviceName  = fs.String("service-name", "annoying.technology", "clear name of the service you want to monitor from render.com dashboard")
		webhookURL   = fs.String("webhook-url", "https://example.com/123/hook", "the url of the webhook we should hit if there was a new deploy")
		deployWindow = fs.Int("deploy-window", 10, "the deploy window in minutes. if within this timeframe there was a successful deploy we call the webhook")
		interval     = fs.String("check-interval", "* * * * *", "how often we check the render api for new deploys. cron syntax.")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
		ff.WithEnvVars(),
	)

	l := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	switch strings.ToLower(*environment) {
	case "development":
		l = level.NewFilter(l, level.AllowInfo())
	case "prod":
		l = level.NewFilter(l, level.AllowError())
	}
	l = log.With(l, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	if *apiToken == "" {
		level.Error(l).Log("err", "api-token has to  be set")
		return
	}

	c := http.DefaultClient
	rs := rendercom.NewRenderService(c, *apiToken)
	s, err := rs.GetServices()
	if err != nil {
		level.Error(l).Log("err", err)
		return
	}

	jobs := cron.New()
	for _, service := range s {
		if service.Service.Suspended != "not_suspended" {
			continue
		}
		if service.Service.Name != *serviceName {
			continue
		}
		level.Info(l).Log("msg", "checking service", "service_name", service.Service.Name)
		_, err = jobs.AddJob(*interval, NewDeployCheckRun(l, c, rs, service.Service.Id, *webhookURL, *deployWindow))
		if err != nil {
			level.Error(l).Log("err", err)
			return
		}
		level.Info(l).Log("msg", "added cronjob", "service_name", service.Service.Name)
	}

	jobs.Start()

	// Set up HTTP API
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("render-deploy-webhook-sender"))
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	level.Info(l).Log("msg", fmt.Sprintf("render-deploy-webhook-sender is running on :%s", *port), "environment", *environment)

	// Set up webserver and and set max file limit to 50MB
	err = http.ListenAndServe(fmt.Sprintf(":%s", *port), &maxBytesHandler{h: r, n: (50 * 1024 * 1024)})
	if err != nil {
		level.Error(l).Log("err", err)
		return
	}
}

type maxBytesHandler struct {
	h http.Handler
	n int64
}

func (h *maxBytesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.n)
	h.h.ServeHTTP(w, r)
}

type DeployCheckRun struct {
	l            log.Logger
	rs           rendercom.RenderService
	c            *http.Client
	serviceID    string
	webhookURL   string
	deployWindow int
}

func NewDeployCheckRun(
	l log.Logger,
	c *http.Client,
	rs rendercom.RenderService,
	serviceId string,
	webhookURL string,
	deployWindow int,
) *DeployCheckRun {
	return &DeployCheckRun{
		l:            l,
		c:            c,
		rs:           rs,
		serviceID:    serviceId,
		webhookURL:   webhookURL,
		deployWindow: deployWindow,
	}
}

func (dc *DeployCheckRun) Run() {
	d, err := dc.rs.GetDeploys(dc.serviceID, "live")
	if err != nil {
		level.Error(dc.l).Log("err", err)
		return
	}
	// There should only be one "live" deploy at any given time
	if len(d) != 1 {
		level.Error(dc.l).Log("msg", "there are more than one deploy live, this should not happen")
		return
	}
	finishedAt := d[0].Deploy.FinishedAt
	//finishedAt = time.Now().Add(time.Duration(-2) * time.Minute)
	if finishedAt.After(time.Now().Add(time.Duration(-dc.deployWindow) * time.Minute)) {
		level.Info(dc.l).Log("msg", "found deploy within deploy window", "deploy_ts", finishedAt, "now_ts", time.Now())
		resp, err := dc.c.Post(dc.webhookURL, "application/json", nil)
		if err != nil {
			level.Error(dc.l).Log("err", err)
			return
		}
		if resp.StatusCode == http.StatusAccepted {
			level.Info(dc.l).Log("msg", "webhook hit successfully", "status_code", resp.StatusCode)
			return
		} else {
			level.Error(dc.l).Log("msg", "unexpected status code", "status_code", resp.StatusCode)
			return
		}
	}
}
