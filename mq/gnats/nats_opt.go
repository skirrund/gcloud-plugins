package gnats

import "time"

type Option struct {
	Url                 string        `property:"nats.url"`
	User                string        `property:"nats.user"`
	Password            string        `property:"nats.password"`
	AppName             string        `property:"nats.appName"`
	Timeout             time.Duration `property:"nats.timeout"`
	MaxPingsOutstanding int           `property:"nats.maxPingsOutstanding"`
}
