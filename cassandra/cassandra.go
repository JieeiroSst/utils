package cassandra

import (
	"time"

	"github.com/gocql/gocql"
)

type Cassandra struct {
	Dns      string
	Username string
	Password string
}

func NewCassandra(c Cassandra) *gocql.Session {
	cluster := gocql.NewCluster(c.Dns)
	cluster.Consistency = gocql.Quorum
	cluster.ProtoVersion = 4
	cluster.ConnectTimeout = time.Second * 10
	cluster.Authenticator = gocql.PasswordAuthenticator{Username: c.Username, Password: c.Password, AllowedAuthenticators: []string{"com.instaclustr.cassandra.auth.InstaclustrPasswordAuthenticator"}}
	session, err := cluster.CreateSession()
	if err != nil {
		return nil
	}
	defer session.Close()
	return session
}
