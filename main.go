package main

import (
	"github.com/urfave/cli"
	"os"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"github.com/orange-cloudfoundry/gobis/handlers"
	log "github.com/sirupsen/logrus"
	"net/http"
	"fmt"
	"strings"
	"github.com/orange-cloudfoundry/gobis/proxy"
	"github.com/orange-cloudfoundry/gobis/middlewares"
	"strconv"
)

func main() {
	app := cli.NewApp()
	app.Name = "gobis"
	app.Version = "1.0.0"
	app.Usage = "Create a gobis server based on a config file"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "gobis-config.yml",
			Usage: "Path to the config file",
		},
		cli.StringFlag{
			Name:  "log-level, l",
			Value: "info",
			Usage: "Log level to use",
		},
		cli.BoolFlag{
			Name:  "log-json, j",
			Usage: "Write log in json",
		},
		cli.BoolFlag{
			Name:  "no-color",
			Usage: "Logger will not display colors",
		},
	}
	app.Action = runServer
	app.Run(os.Args)
}
func loadLogConfig(c *cli.Context) {
	noColor := c.GlobalBool("no-color")
	logJson := c.GlobalBool("log-json")
	if logJson {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			DisableColors: noColor,
		})
	}

	logLevel := c.GlobalString("log-level")
	if logLevel == "" {
		return
	}
	switch strings.ToUpper(logLevel) {
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
		return
	case "WARN":
		log.SetLevel(log.WarnLevel)
		return
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
		return
	case "PANIC":
		log.SetLevel(log.PanicLevel)
		return
	case "FATAL":
		log.SetLevel(log.FatalLevel)
		return
	}

	return
}
func runServer(c *cli.Context) error {
	loadLogConfig(c)
	configPath := c.GlobalString("config")
	conf, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	gobisHandler, err := handlers.NewDefaultHandlerWithRouterFactory(
		conf,
		proxy.NewRouterFactory(
			middlewares.BasicAuth,
			middlewares.RateLimit,
			middlewares.ConnLimit,
			middlewares.Cors,
			middlewares.Trace,
		),
	)
	if err != nil {
		return err
	}
	servAddr := gobisHandler.GetServerAddr()
	log.Infof("Serving gobis server on address '%s'", servAddr)
	return http.ListenAndServe(servAddr, gobisHandler)
}
func loadConfig(path string) (handlers.DefaultHandlerConfig, error) {
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return handlers.DefaultHandlerConfig{}, err
	}
	conf := handlers.DefaultHandlerConfig{}
	err = yaml.Unmarshal(dat, &conf)
	if err != nil {
		return handlers.DefaultHandlerConfig{}, err
	}
	if conf.Port == 0 {
		port, _ := strconv.Atoi(os.Getenv("PORT"))
		conf.Port = port
	}
	if len(conf.Routes) == 0 {
		return conf, fmt.Errorf("You must configure routes in your config file")
	}
	return conf, nil
}