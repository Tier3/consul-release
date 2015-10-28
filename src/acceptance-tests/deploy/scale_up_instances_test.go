package deploy_test

import (
	"acceptance-tests/helpers"

	capi "github.com/hashicorp/consul/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Scaling up Instances", func() {
	var (
		consulManifest  *helpers.Manifest
		consulServerIPs []string
		runner          *helpers.AgentRunner
	)

	BeforeEach(func() {
		consulManifest = new(helpers.Manifest)
		consulServerIPs = []string{}
	})

	AfterEach(func() {
		By("delete deployment")
		runner.Stop()
		bosh.Command("-n", "delete", "deployment", consulDeployment)
	})

	Describe("scaling from 3 nodes to 5", func() {
		It("succesfully scales to more consul nodes, persisting data", func() {
			By("deploying 3 nodes")
			bosh.GenerateAndSetDeploymentManifest(
				consulManifest,
				consulManifestGeneration,
				directorUUIDStub,
				helpers.InstanceCount3NodesStubPath,
				helpers.PersistentDiskStubPath,
				config.IAASSettingsConsulStubPath,
				helpers.PropertyOverridesStubPath,
				consulNameOverrideStub,
			)
			Expect(bosh.Command("-n", "deploy")).To(gexec.Exit(0))
			Expect(len(consulManifest.Properties.Consul.Agent.Servers.Lans)).To(Equal(3))

			for _, elem := range consulManifest.Properties.Consul.Agent.Servers.Lans {
				consulServerIPs = append(consulServerIPs, elem)
			}

			runner = helpers.NewAgentRunner(consulServerIPs, config.BindAddress)
			runner.Start()

			By("setting a persistent value")
			consatsClient := runner.NewClient()

			consatsKey := "consats-key"
			consatsValue := []byte("consats-value")

			keyValueClient := consatsClient.KV()

			pair := &capi.KVPair{Key: consatsKey, Value: consatsValue}
			_, err := keyValueClient.Put(pair, nil)
			Expect(err).ToNot(HaveOccurred())

			resultPair, _, err := keyValueClient.Get(consatsKey, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultPair.Value).To(Equal(consatsValue))

			By("scaling up to 5 nodes")
			bosh.GenerateAndSetDeploymentManifest(
				consulManifest,
				consulManifestGeneration,
				directorUUIDStub,
				helpers.InstanceCount5NodesStubPath,
				helpers.PersistentDiskStubPath,
				config.IAASSettingsConsulStubPath,
				helpers.PropertyOverridesStubPath,
				consulNameOverrideStub,
			)

			Expect(bosh.Command("-n", "deploy")).To(gexec.Exit(0))
			Expect(len(consulManifest.Properties.Consul.Agent.Servers.Lans)).To(Equal(5))

			By("reading the value from consul")
			resultPair, _, err = keyValueClient.Get(consatsKey, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultPair).NotTo(BeNil())
			Expect(resultPair.Value).To(Equal(consatsValue))
		})
	})

	Describe("checking data persistence when scaling from 1 node to 3", func() {
		It("succesfully scales from 1 to multiple consul nodes", func() {
			By("deploying 1 node")
			bosh.GenerateAndSetDeploymentManifest(
				consulManifest,
				consulManifestGeneration,
				directorUUIDStub,
				helpers.InstanceCount1NodeStubPath,
				helpers.PersistentDiskStubPath,
				config.IAASSettingsConsulStubPath,
				helpers.PropertyOverridesStubPath,
				consulNameOverrideStub,
			)
			Expect(bosh.Command("-n", "deploy")).To(gexec.Exit(0))
			Expect(len(consulManifest.Properties.Consul.Agent.Servers.Lans)).To(Equal(1))

			By("starting local consul agent")
			for _, elem := range consulManifest.Properties.Consul.Agent.Servers.Lans {
				consulServerIPs = append(consulServerIPs, elem)
			}

			runner = helpers.NewAgentRunner(consulServerIPs, config.BindAddress)
			runner.Start()

			By("writing the value to consul")
			consatsClient := runner.NewClient()

			consatsKey := "consats-key"
			consatsValue := []byte("consats-value")

			keyValueClient := consatsClient.KV()
			pair := &capi.KVPair{Key: consatsKey, Value: consatsValue}
			_, err := keyValueClient.Put(pair, nil)
			Expect(err).ToNot(HaveOccurred())

			By("scaling up to 3 nodes")
			bosh.GenerateAndSetDeploymentManifest(
				consulManifest,
				consulManifestGeneration,
				directorUUIDStub,
				helpers.InstanceCount3NodesStubPath,
				helpers.PersistentDiskStubPath,
				config.IAASSettingsConsulStubPath,
				helpers.PropertyOverridesStubPath,
				consulNameOverrideStub,
			)

			Expect(bosh.Command("-n", "deploy")).To(gexec.Exit(0))
			Expect(len(consulManifest.Properties.Consul.Agent.Servers.Lans)).To(Equal(3))

			By("reading the value from consul")
			resultPair, _, err := keyValueClient.Get(consatsKey, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultPair).ToNot(BeNil())
			Expect(resultPair.Value).To(Equal(consatsValue))
		})
	})
})
