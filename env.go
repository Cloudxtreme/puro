package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
)

var (
	etcc *etcd.Client
	log  *logrus.Logger
)
