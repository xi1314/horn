package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hiwjd/horn/consumer/dispatcher"
	"github.com/hiwjd/horn/state/remote"

	"github.com/BurntSushi/toml"
	"github.com/nsqio/go-nsq"
)

var (
	configPath string
	config     dispatcher.Config
)

func init() {
	flag.StringVar(&configPath, "c", "./dispatcher.toml", "配置文件的路径")
}

func main() {
	flag.Parse()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("配置文件 %s 不存在, 使用默认配置 \r\n", configPath)
	} else {
		_, err := toml.DecodeFile(configPath, &config)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println(config)

	cfg := nsq.NewConfig()

	if config.Channel == "" {
		log.Fatal("--channel is required")
	}

	if len(config.Topics) < 1 {
		log.Fatal("--topic is required")
	}

	if len(config.NsqdTCPAddrs) == 0 && len(config.LookupdHTTPAddrs) == 0 {
		log.Fatal("--nsqd-tcp-address or --lookupd-http-address required")
	}
	if len(config.NsqdTCPAddrs) > 0 && len(config.LookupdHTTPAddrs) > 0 {
		log.Fatal("use --nsqd-tcp-address or --lookupd-http-address not both")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cfg.UserAgent = "useragent"
	cfg.MaxInFlight = config.MaxInFlight

	//redisManager := redis.New(config.RedisConfigs)
	//mysqlManager := mysql.New(config.MysqlConfigs)
	//s := state.New(mysqlManager, redisManager)
	s := remote.New("http://127.0.0.1:9094")
	handler := dispatcher.NewHandler(s)

	consumers := make(map[*nsq.Consumer]int, len(config.Topics))
	consumerStoped := make(chan *nsq.Consumer)

	for _, topic := range config.Topics {
		consumer, err := nsq.NewConsumer(topic, config.Channel, cfg)
		if err != nil {
			log.Fatal(err)
		}

		consumer.AddConcurrentHandlers(handler, 4)

		err = consumer.ConnectToNSQDs(config.NsqdTCPAddrs)
		if err != nil {
			log.Fatal(err)
		}

		err = consumer.ConnectToNSQLookupds(config.LookupdHTTPAddrs)
		if err != nil {
			log.Fatal(err)
		}

		consumers[consumer] = 1

		go func(consumer *nsq.Consumer) {
			select {
			case <-consumer.StopChan:
				consumerStoped <- consumer
				return
			}
		}(consumer)
	}

	for {
		select {
		case consumer := <-consumerStoped:
			delete(consumers, consumer)
			if len(consumers) == 0 {
				return
			}
		case <-sigChan:
			for consumer, _ := range consumers {
				consumer.Stop()
			}
		}
	}
}
