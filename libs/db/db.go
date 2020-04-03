package db

import (
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("test")

var (
	cassTimeout        int = 600 // Second
	cassConnectTimeout int = 600 // Second
	session            *gocql.Session
)

// CassConfig config for cassandra
type CassConfig struct {
	Hosts    string
	Username string
	Password string
	Keyspace string
	Port     int
}

func connectCluster(cf *CassConfig) *gocql.ClusterConfig {
	// connect to the cluster
	logger.Infof("Connect cassandra cluster:%+v", *cf)
	cluster := gocql.NewCluster(cf.Hosts)
	cluster.Port = cf.Port
	cluster.Keyspace = cf.Keyspace
	cluster.Timeout = time.Duration(cassTimeout) * time.Second
	cluster.ConnectTimeout = time.Duration(cassConnectTimeout) * time.Second
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cf.Username,
		Password: cf.Password,
	}
	cluster.Consistency = gocql.LocalQuorum
	cluster.NumConns = 3 // set connection pool num
	return cluster
}

// NewSession return the cassandra session
func NewSession(cf *CassConfig) (*gocql.Session, error) {
	cassCluster := connectCluster(cf)
	return cassCluster.CreateSession()
}

// NewSessionWithRetry return the cassandra session
func NewSessionWithRetry(cf *CassConfig) (*gocql.Session, error) {
	if session != nil {
		return session, nil
	}
	interval := time.Duration(15)
	timeout := time.NewTimer(30 * time.Minute)
	var err error

loop:
	for {
		session, err = NewSession(cf)
		if err == nil && session != nil {
			break loop
		}
		logger.Warningf("new cassandra session failed, %v", err)

		// retry or timeout
		select {
		case <-time.After(interval * time.Second):
			logger.Infof("retry new cassandra session after %d second", interval)
		case <-timeout.C:
			err = fmt.Errorf("new cassandra session failed after retry many times, cause by %v", err)
			break loop
		}
	}
	return session, err
}

// Execute ...
func Execute(session *gocql.Session, cmd string) error {
	if err := session.Query(cmd).Exec(); err != nil {
		logger.Error(err)
		return err
	}
	return nil
}

// TruncateTable ...
func TruncateTable(session *gocql.Session, table string) error {
	cmd := "TRUNCATE " + table
	return Execute(session, cmd)
}
