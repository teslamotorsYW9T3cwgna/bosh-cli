package vm_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	bmdepl "github.com/cloudfoundry/bosh-micro-cli/deployment"
	bmeventlog "github.com/cloudfoundry/bosh-micro-cli/eventlogging"
	bmstemcell "github.com/cloudfoundry/bosh-micro-cli/stemcell"

	fakebmlog "github.com/cloudfoundry/bosh-micro-cli/eventlogging/fakes"
	fakebmvm "github.com/cloudfoundry/bosh-micro-cli/vm/fakes"

	. "github.com/cloudfoundry/bosh-micro-cli/vm"
)

var _ = Describe("Manager", func() {
	Describe("CreateVM", func() {
		var (
			infrastructure          *fakebmvm.FakeInfrastructure
			eventLogger             *fakebmlog.FakeEventLogger
			manager                 Manager
			expectedStemcellCID     bmstemcell.CID
			expectedVMCID           CID
			expectedNetworksSpec    map[string]interface{}
			expectedCloudProperties map[string]interface{}
			expectedEnv             map[string]interface{}
			stemcellCID             bmstemcell.CID
			deployment              bmdepl.Deployment
		)

		BeforeEach(func() {
			infrastructure = fakebmvm.NewFakeInfrastructure()
			eventLogger = fakebmlog.NewFakeEventLogger()
			manager = NewManagerFactory(eventLogger).NewManager(infrastructure)
			expectedStemcellCID = bmstemcell.CID("fake-stemcell-cid")
			expectedVMCID = CID("fake-vm-cid")
			expectedNetworksSpec = map[string]interface{}{
				"fake-network-name": map[string]interface{}{
					"type":             "dynamic",
					"cloud_properties": map[string]interface{}{},
				},
			}
			expectedCloudProperties = map[string]interface{}{
				"fake-cloud-property-key": "fake-cloud-property-value",
			}
			expectedEnv = map[string]interface{}{
				"fake-env-key": "fake-env-value",
			}
			infrastructure.SetCreateVMBehavior(expectedVMCID, nil)
			stemcellCID = bmstemcell.CID("fake-stemcell-cid")
			deployment = bmdepl.Deployment{
				Name: "fake-deployment",
				Networks: []bmdepl.Network{
					{
						Name: "fake-network-name",
						Type: "dynamic",
					},
				},
				ResourcePools: []bmdepl.ResourcePool{
					{
						Name: "fake-resource-pool-name",
						RawCloudProperties: map[interface{}]interface{}{
							"fake-cloud-property-key": "fake-cloud-property-value",
						},
						RawEnv: map[interface{}]interface{}{
							"fake-env-key": "fake-env-value",
						},
					},
				},
			}
		})

		It("creates a VM", func() {
			vmCID, err := manager.CreateVM(expectedStemcellCID, deployment)
			Expect(err).ToNot(HaveOccurred())
			Expect(vmCID).To(Equal(expectedVMCID))
			Expect(infrastructure.CreateInput).To(Equal(
				fakebmvm.CreateInput{
					StemcellCID:     expectedStemcellCID,
					CloudProperties: expectedCloudProperties,
					NetworksSpec:    expectedNetworksSpec,
					Env:             expectedEnv,
				},
			))
		})

		It("logs start and stop events to the eventLogger", func() {
			_, err := manager.CreateVM(stemcellCID, deployment)
			Expect(err).ToNot(HaveOccurred())

			expectedStartEvent := bmeventlog.Event{
				Stage: "Deploy Micro BOSH",
				Total: 1,
				Task:  fmt.Sprintf("Creating VM from %s", expectedStemcellCID),
				Index: 1,
				State: bmeventlog.Started,
			}

			expectedFinishEvent := bmeventlog.Event{
				Stage: "Deploy Micro BOSH",
				Total: 1,
				Task:  fmt.Sprintf("Creating VM from %s", expectedStemcellCID),
				Index: 1,
				State: bmeventlog.Finished,
			}

			Expect(eventLogger.LoggedEvents).To(ContainElement(expectedStartEvent))
			Expect(eventLogger.LoggedEvents).To(ContainElement(expectedFinishEvent))
			Expect(eventLogger.LoggedEvents).To(HaveLen(2))
		})

		Context("when creating the vm fails", func() {
			It("logs start and failure events to the eventLogger", func() {
				infrastructure.SetCreateVMBehavior(expectedVMCID, bosherr.New("fake-create-error"))

				_, err := manager.CreateVM(stemcellCID, deployment)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-create-error"))

				expectedStartEvent := bmeventlog.Event{
					Stage: "Deploy Micro BOSH",
					Total: 1,
					Task:  fmt.Sprintf("Creating VM from %s", expectedStemcellCID),
					Index: 1,
					State: bmeventlog.Started,
				}

				expectedFailedEvent := bmeventlog.Event{
					Stage:   "Deploy Micro BOSH",
					Total:   1,
					Task:    fmt.Sprintf("Creating VM from %s", expectedStemcellCID),
					Index:   1,
					State:   bmeventlog.Failed,
					Message: "fake-create-error",
				}

				Expect(eventLogger.LoggedEvents).To(ContainElement(expectedStartEvent))
				Expect(eventLogger.LoggedEvents).To(ContainElement(expectedFailedEvent))
				Expect(eventLogger.LoggedEvents).To(HaveLen(2))
			})
		})
	})
})