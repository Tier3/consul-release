package main_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func killProcessWithPIDFile(pidFilePath string) {
	pidFileContents, err := ioutil.ReadFile(pidFilePath)
	Expect(err).NotTo(HaveOccurred())

	pid, err := strconv.Atoi(string(pidFileContents))
	Expect(err).NotTo(HaveOccurred())

	process, err := os.FindProcess(pid)
	Expect(err).NotTo(HaveOccurred())

	process.Signal(syscall.SIGKILL)
}

var _ = Describe("confab", func() {
	var (
		consulConfigDir string
		pidFile         *os.File
	)

	BeforeEach(func() {
		var err error
		consulConfigDir, err = ioutil.TempDir("", "fake-agent-config-dir")
		Expect(err).NotTo(HaveOccurred())

		pidFile, err = ioutil.TempFile("", "fake-pid-file")
		Expect(err).NotTo(HaveOccurred())

		options := []byte(`{ "RunClient": true, "Members": ["member-1", "member-2", "member-3"] }`)
		Expect(ioutil.WriteFile(filepath.Join(consulConfigDir, "options.json"), options, 0600)).To(Succeed())
	})

	Context("when starting", func() {
		Context("for a client", func() {
			It("starts a consul agent as a client", func() {
				cmd := exec.Command(pathToConfab,
					"start",
					"--server=false",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir,
					"--expected-member", "member-1",
					"--expected-member", "member-2",
					"--expected-member", "member-3",
				)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(0))

				pidFileContents, err := ioutil.ReadFile(pidFile.Name())
				Expect(err).NotTo(HaveOccurred())

				pid, err := strconv.Atoi(string(pidFileContents))
				Expect(err).NotTo(HaveOccurred())

				fakeOutput, err := ioutil.ReadFile(filepath.Join(consulConfigDir, "fake-output.json"))
				Expect(err).NotTo(HaveOccurred())

				var decodedFakeOutput map[string]interface{}
				err = json.Unmarshal(fakeOutput, &decodedFakeOutput)
				Expect(err).NotTo(HaveOccurred())

				Expect(decodedFakeOutput).To(Equal(map[string]interface{}{
					"PID": float64(pid),
					"Args": []interface{}{
						"agent",
						fmt.Sprintf("-config-dir=%s", consulConfigDir),
					},
				}))
			})
		})

		Context("for a server", func() {
			var (
				session *gexec.Session
			)
			BeforeEach(func() {
				options := []byte(`{ "RunServer": true, "Members": ["member-1", "member-2", "member-3"] }`)
				Expect(ioutil.WriteFile(filepath.Join(consulConfigDir, "options.json"), options, 0600)).To(Succeed())
			})
			AfterEach(func() {
				killProcessWithPIDFile(pidFile.Name())
				if session.Command == nil || session.Command.Process == nil {
					return
				}
				session.Kill()
			})

			It("starts a consul agent as a server", func() {
				cmd := exec.Command(pathToConfab,
					"start",
					"--server=true",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir,
					"--expected-member", "member-1",
					"--expected-member", "member-2",
					"--expected-member", "member-3",
					"--encryption-key", "key-1",
					"--encryption-key", "key-2",
				)
				var err error
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(0))

				pidFileContents, err := ioutil.ReadFile(pidFile.Name())
				Expect(err).NotTo(HaveOccurred())

				pid, err := strconv.Atoi(string(pidFileContents))
				Expect(err).NotTo(HaveOccurred())

				fakeOutput, err := ioutil.ReadFile(filepath.Join(consulConfigDir, "fake-output.json"))
				Expect(err).NotTo(HaveOccurred())

				var decodedFakeOutput map[string]interface{}
				err = json.Unmarshal(fakeOutput, &decodedFakeOutput)
				Expect(err).NotTo(HaveOccurred())

				Expect(decodedFakeOutput).To(Equal(map[string]interface{}{
					"PID": float64(pid),
					"Args": []interface{}{
						"agent",
						fmt.Sprintf("-config-dir=%s", consulConfigDir),
					},
				}))

				Expect(session.Out).To(gbytes.Say("UseKey called"))
			})
		})
	})

	FContext("when stopping", func() {

		BeforeEach(func() {
			options := []byte(`{ "RunServer": true, "Members": ["member-1", "member-2", "member-3"], "WaitForHUP": true }`)
			Expect(ioutil.WriteFile(filepath.Join(consulConfigDir, "options.json"), options, 0600)).To(Succeed())
		})

		It("stops the consul agent", func() {
			cmd := exec.Command(pathToConfab,
				"start",
				"--server=true",
				"--pid-file", pidFile.Name(),
				"--agent-path", pathToFakeAgent,
				"--consul-config-dir", consulConfigDir,
				"--expected-member", "member-1",
				"--expected-member", "member-2",
				"--expected-member", "member-3",
			)
			serverSession, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(15 * time.Second)

			cmd = exec.Command(pathToConfab,
				"stop",
				"--pid-file", pidFile.Name(),
				"--agent-path", pathToFakeAgent,
				"--consul-config-dir", consulConfigDir,
				"--expected-member", "member-1",
				"--expected-member", "member-2",
				"--expected-member", "member-3",
			)

			stopSession, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(stopSession, "5s").Should(gexec.Exit(0))
			Eventually(serverSession, "5s").Should(gexec.Exit(0))

			pidFileContents, err := ioutil.ReadFile(pidFile.Name())
			Expect(err).NotTo(HaveOccurred())

			pid, err := strconv.Atoi(string(pidFileContents))
			Expect(err).NotTo(HaveOccurred())

			fakeOutput, err := ioutil.ReadFile(filepath.Join(consulConfigDir, "fake-output.json"))
			Expect(err).NotTo(HaveOccurred())

			var decodedFakeOutput map[string]interface{}
			err = json.Unmarshal(fakeOutput, &decodedFakeOutput)
			Expect(err).NotTo(HaveOccurred())

			Expect(decodedFakeOutput).To(Equal(map[string]interface{}{
				"PID": float64(pid),
				"Args": []interface{}{
					"agent",
					fmt.Sprintf("-config-dir=%s", consulConfigDir),
				},
			}))

			Expect(serverSession.Out).To(gbytes.Say("Leave called"))
		})
	})

	Context("failure cases", func() {
		Context("when no arguments are provided", func() {
			It("returns a non-zero status code and prints usage", func() {
				cmd := exec.Command(pathToConfab)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(1))

				Expect(session.Err.Contents()).To(ContainSubstring("invalid number of arguments"))

				usageLines := []string{
					"usage: confab COMMAND OPTIONS",
					"COMMAND: \"start\" or \"stop\"",
					"-agent-path executable",
					"path to the on-filesystem consul executable",
					"-consul-config-dir directory",
					"path to consul configuration directory",
					"-expected-member list",
					"address list of the expected members",
					"-pid-file file",
					"path to consul PID file",
				}
				for _, line := range usageLines {
					Expect(session.Err.Contents()).To(ContainSubstring(line))
				}
			})
		})

		Context("when no command is provided", func() {
			It("returns a non-zero status code and prints usage", func() {
				cmd := exec.Command(pathToConfab,
					"--server=false",
				)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(1))
				Expect(session.Err.Contents()).To(ContainSubstring("invalid COMMAND \"--server=false\""))
				Expect(session.Err.Contents()).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when an invalid command is provided", func() {
			It("returns a non-zero status code and prints usage", func() {
				cmd := exec.Command(pathToConfab, "banana",
					"--agent-path", pathToFakeAgent,
					"--pid-file", pidFile.Name(),
					"--consul-config-dir", consulConfigDir)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(1))
				Expect(session.Err.Contents()).To(ContainSubstring("invalid COMMAND \"banana\""))
				Expect(session.Err.Contents()).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("expected-member is missing", func() {
			It("prints an error and usage", func() {
				cmd := exec.Command(pathToConfab, "start",
					"--agent-path", pathToFakeAgent,
					"--pid-file", pidFile.Name(),
					"--consul-config-dir", consulConfigDir)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(1))
				Expect(session.Err.Contents()).To(ContainSubstring("at least one \"expected-member\" must be provided"))
				Expect(session.Err.Contents()).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when the agent executable does not exist", func() {
			It("prints an error and usage", func() {
				cmd := exec.Command(pathToConfab, "start",
					"--expected-member", "member-1",
					"--agent-path", "/tmp/path/that/does/not/exist",
					"--pid-file", pidFile.Name(),
					"--consul-config-dir", consulConfigDir)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(1))
				Expect(session.Err.Contents()).To(ContainSubstring("\"agent-path\" \"/tmp/path/that/does/not/exist\" cannot be found"))
				Expect(session.Err.Contents()).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when the PID file option is not provided", func() {
			It("prints an error and usage", func() {
				cmd := exec.Command(pathToConfab, "start",
					"--expected-member", "member-1",
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(1))
				Expect(session.Err.Contents()).To(ContainSubstring("\"pid-file\" cannot be empty"))
				Expect(session.Err.Contents()).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when the consul config dir is not provided", func() {
			It("prints an error and usage", func() {
				cmd := exec.Command(pathToConfab, "start",
					"--expected-member", "member-1",
					"--agent-path", pathToFakeAgent,
					"--pid-file", pidFile.Name(),
					"--consul-config-dir", "/tmp/path/that/does/not/exist")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(1))
				Expect(session.Err.Contents()).To(ContainSubstring("\"consul-config-dir\" \"/tmp/path/that/does/not/exist\" could not be found"))
				Expect(session.Err.Contents()).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when the pid file contains the pid of a running process", func() {
			It("prints an error and exits status 1", func() {
				// some test setup....
				myPID := os.Getpid()
				Expect(ioutil.WriteFile(pidFile.Name(), []byte(fmt.Sprintf("%d", myPID)), 0644)).To(Succeed())

				cmd := exec.Command(pathToConfab,
					"start",
					"--server=false",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir,
					"--expected-member", "member-1",
					"--expected-member", "member-2",
					"--expected-member", "member-3",
				)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(1))

				Expect(session.Err.Contents()).To(ContainSubstring("error booting consul agent"))
				Expect(session.Err.Contents()).To(ContainSubstring("already running"))
			})
		})

		Context("when the rpc connection cannot be created", func() {
			It("returns an error and exits with status 1", func() {

				options := []byte(`{ "RunClient": true, "Members": ["member-1", "member-2", "member-3"], "FailRPCServer": true }`)
				Expect(ioutil.WriteFile(filepath.Join(consulConfigDir, "options.json"), options, 0600)).To(Succeed())

				cmd := exec.Command(pathToConfab,
					"start",
					"--server=true",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir,
					"--expected-member", "member-1",
					"--expected-member", "member-2",
					"--expected-member", "member-3",
				)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(1))

				Expect(session.Err.Contents()).To(ContainSubstring("error connecting to RPC server"))
			})
		})
	})
})
