package confab

import (
	"errors"

	"github.com/hashicorp/consul/api"
)

type consulAPIAgent interface {
	Members(wan bool) ([]*api.AgentMember, error)
}

type consulRPCClient interface {
	Stats() (map[string]map[string]string, error)
}

type AgentClient struct {
	ExpectedMembers []string
	ConsulAPIAgent  consulAPIAgent
	ConsulRPCClient consulRPCClient
}

/*

map[string]map[string]string{
	"raft": map[string]string{
		"commit_index": "x",
		"last_log_index": "y",
	}
}

*/

func (c AgentClient) VerifyJoined() error {
	members, err := c.ConsulAPIAgent.Members(false)
	if err != nil {
		return err
	}

	for _, member := range members {
		for _, expectedMember := range c.ExpectedMembers {
			if member.Addr == expectedMember {
				return nil
			}
		}
	}

	return errors.New("no expected members")
}

func (c AgentClient) VerifySynced() error {
	stats, err := c.ConsulRPCClient.Stats()
	if err != nil {
		return err
		panic(err)
	}

	if stats["raft"]["commit_index"] != stats["raft"]["last_log_index"] {
		return errors.New("Log not in sync")
	}

	if stats["raft"]["commit_index"] == "0" {
		return errors.New("Commit index must not be zero")
	}

	return nil
}
