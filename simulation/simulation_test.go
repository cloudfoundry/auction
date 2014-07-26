package simulation_test

import (
	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/simulation/visualization"
	"github.com/cloudfoundry-incubator/auction/util"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Ω

var _ = Describe("Auction", func() {
	var initialDistributions map[int][]auctiontypes.SimulatedInstance

	newSimulatedInstance := func(processGuid string, index int, memoryMB int) auctiontypes.SimulatedInstance {
		return auctiontypes.SimulatedInstance{
			ProcessGuid:  processGuid,
			InstanceGuid: util.NewGuid("INS"),
			Index:        index,
			MemoryMB:     memoryMB,
			DiskMB:       1,
		}
	}

	generateUniqueSimulatedInstances := func(numInstances int, index int, memoryMB int) []auctiontypes.SimulatedInstance {
		instances := []auctiontypes.SimulatedInstance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newSimulatedInstance(util.NewGrayscaleGuid("AAA"), index, memoryMB))
		}
		return instances
	}

	generateSimulatedInstancesForProcessGuid := func(processGuid string, numInstances int, index int, memoryMB int) []auctiontypes.SimulatedInstance {
		instances := []auctiontypes.SimulatedInstance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newSimulatedInstance(processGuid, index, memoryMB))
		}
		return instances
	}

	newLRPStartAuction := func(processGuid string, memoryMB int) models.LRPStartAuction {
		return models.LRPStartAuction{
			ProcessGuid:  processGuid,
			InstanceGuid: util.NewGuid("INS"),
			MemoryMB:     memoryMB,
			DiskMB:       1,
			Index:        0,
		}
	}

	generateUniqueLRPStartAuctions := func(numInstances int, memoryMB int) []models.LRPStartAuction {
		instances := []models.LRPStartAuction{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newLRPStartAuction(util.NewGrayscaleGuid("BBB"), memoryMB))
		}
		return instances
	}

	randomSVGColor := func() string {
		return []string{"purple", "red", "cyan", "teal", "gray", "blue", "pink", "green", "lime", "orange", "lightseagreen", "brown"}[util.R.Intn(12)]
	}

	generateLRPStartAuctionsWithRandomSVGColors := func(numInstances int, memoryMB int) []models.LRPStartAuction {
		instances := []models.LRPStartAuction{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newLRPStartAuction(randomSVGColor(), memoryMB))
		}
		return instances
	}

	generateLRPStartAuctionsForProcessGuid := func(numInstances int, processGuid string, memoryMB int) []models.LRPStartAuction {
		instances := []models.LRPStartAuction{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newLRPStartAuction(processGuid, memoryMB))
		}
		return instances
	}

	BeforeEach(func() {
		util.ResetGuids()
		initialDistributions = map[int][]auctiontypes.SimulatedInstance{}
	})

	JustBeforeEach(func() {
		for index, simulatedInstances := range initialDistributions {
			client.SetSimulatedInstances(repGuids[index], simulatedInstances)
		}
	})

	Describe("Experiments", func() {
		Context("Cold start scenario", func() {
			nexec := []int{25, 100}
			n1apps := []int{1800, 7000}
			n2apps := []int{200, 1000}
			n4apps := []int{50, 200}
			for i := range nexec {
				i := i
				Context("with single-instance and multi-instance apps", func() {
					It("should distribute evenly", func() {
						instances := []models.LRPStartAuction{}

						instances = append(instances, generateUniqueLRPStartAuctions(n1apps[i]/2, 1)...)
						instances = append(instances, generateLRPStartAuctionsWithRandomSVGColors(n1apps[i]/2, 1)...)
						instances = append(instances, generateUniqueLRPStartAuctions(n2apps[i]/2, 2)...)
						instances = append(instances, generateLRPStartAuctionsWithRandomSVGColors(n2apps[i]/2, 2)...)
						instances = append(instances, generateUniqueLRPStartAuctions(n4apps[i]/2, 4)...)
						instances = append(instances, generateLRPStartAuctionsWithRandomSVGColors(n4apps[i]/2, 4)...)

						report := auctionDistributor.HoldAuctionsFor(instances, repGuids[:nexec[i]], auctionrunner.DefaultStartAuctionRules)

						visualization.PrintReport(client, report.AuctionResults, repGuids[:nexec[i]], report.AuctionDuration, auctionrunner.DefaultStartAuctionRules)

						svgReport.DrawReportCard(i, 0, report)
						reports = append(reports, report)
					})
				})
			}

		})

		Context("Imbalanced scenario (e.g. a deploy)", func() {
			nexec := []int{100, 100}
			nempty := []int{5, 1}
			napps := []int{500, 100}

			for i := range nexec {
				i := i
				Context("scenario", func() {
					BeforeEach(func() {
						for j := 0; j < nexec[i]-nempty[i]; j++ {
							initialDistributions[j] = generateUniqueSimulatedInstances(50, 0, 1)
						}
					})

					It("should distribute evenly", func() {
						instances := generateUniqueLRPStartAuctions(napps[i], 1)

						report := auctionDistributor.HoldAuctionsFor(instances, repGuids[:nexec[i]], auctionrunner.DefaultStartAuctionRules)

						visualization.PrintReport(client, report.AuctionResults, repGuids[:nexec[i]], report.AuctionDuration, auctionrunner.DefaultStartAuctionRules)

						svgReport.DrawReportCard(i, 1, report)
						reports = append(reports, report)
					})
				})
			}
		})

		Context("The Watters demo", func() {
			nexec := []int{30, 100}
			napps := []int{200, 400}

			for i := range nexec {
				i := i

				Context("scenario", func() {
					BeforeEach(func() {
						for j := 0; j < nexec[i]; j++ {
							initialDistributions[j] = generateUniqueSimulatedInstances(util.RandomIntIn(78, 80), 0, 1)
						}
					})

					It("should distribute evenly", func() {
						instances := generateLRPStartAuctionsForProcessGuid(napps[i], "red", 1)

						report := auctionDistributor.HoldAuctionsFor(instances, repGuids[:nexec[i]], auctionrunner.DefaultStartAuctionRules)

						visualization.PrintReport(client, report.AuctionResults, repGuids[:nexec[i]], report.AuctionDuration, auctionrunner.DefaultStartAuctionRules)

						svgReport.DrawReportCard(i, 2, report)
						reports = append(reports, report)
					})
				})
			}
		})

		Context("Stop Auctions", func() {
			processGuid := util.NewGrayscaleGuid("AAA")

			Context("when there are duplicate instances on executors with disaparate resource availabilities", func() {
				BeforeEach(func() {
					initialDistributions[0] = generateUniqueSimulatedInstances(50, 0, 1)
					initialDistributions[0] = append(initialDistributions[0], generateSimulatedInstancesForProcessGuid(processGuid, 1, 0, 1)...)

					initialDistributions[1] = generateUniqueSimulatedInstances(30, 0, 1)
					initialDistributions[1] = append(initialDistributions[1], generateSimulatedInstancesForProcessGuid(processGuid, 1, 0, 1)...)
				})

				It("should favor removing the instance from the heavy-laden executor", func() {
					stopAuctions := []models.LRPStopAuction{
						{
							ProcessGuid: processGuid,
							Index:       0,
						},
					}

					results := auctionDistributor.HoldStopAuctions(stopAuctions, repGuids)
					Ω(results).Should(HaveLen(1))
					Ω(results[0].Winner).Should(Equal("REP-2"))

					instancesOn0 := client.SimulatedInstances(repGuids[0])
					instancesOn1 := client.SimulatedInstances(repGuids[1])

					Ω(instancesOn0).Should(HaveLen(50))
					Ω(instancesOn1).Should(HaveLen(31))
				})
			})

			Context("when the executor with more available resources already has another instance of the app running", func() {
				BeforeEach(func() {
					initialDistributions[0] = generateUniqueSimulatedInstances(50, 0, 1)
					initialDistributions[0] = append(initialDistributions[0], generateSimulatedInstancesForProcessGuid(processGuid, 1, 0, 1)...)

					initialDistributions[1] = generateUniqueSimulatedInstances(30, 0, 1)
					initialDistributions[1] = append(initialDistributions[1], generateSimulatedInstancesForProcessGuid(processGuid, 1, 0, 1)...)
					initialDistributions[1] = append(initialDistributions[1], generateSimulatedInstancesForProcessGuid(processGuid, 1, 1, 1)...)
				})

				It("should favor leaving the instance on the more heavy-laden executor", func() {
					stopAuctions := []models.LRPStopAuction{
						{
							ProcessGuid: processGuid,
							Index:       0,
						},
					}

					results := auctionDistributor.HoldStopAuctions(stopAuctions, repGuids)
					Ω(results).Should(HaveLen(1))
					Ω(results[0].Winner).Should(Equal("REP-1"))

					instancesOn0 := client.SimulatedInstances(repGuids[0])
					instancesOn1 := client.SimulatedInstances(repGuids[1])

					Ω(instancesOn0).Should(HaveLen(51))
					Ω(instancesOn1).Should(HaveLen(31))
				})
			})

			Context("when the executor with fewer available resources has two instances running", func() {
				BeforeEach(func() {
					initialDistributions[0] = generateUniqueSimulatedInstances(50, 0, 1)
					initialDistributions[0] = append(initialDistributions[0], generateSimulatedInstancesForProcessGuid(processGuid, 1, 0, 1)...)

					initialDistributions[1] = generateUniqueSimulatedInstances(30, 0, 1)
					initialDistributions[1] = append(initialDistributions[1], generateSimulatedInstancesForProcessGuid(processGuid, 2, 0, 1)...)
				})

				It("should favor removing the instance from the heavy-laden executor", func() {
					stopAuctions := []models.LRPStopAuction{
						{
							ProcessGuid: processGuid,
							Index:       0,
						},
					}

					results := auctionDistributor.HoldStopAuctions(stopAuctions, repGuids)
					Ω(results).Should(HaveLen(1))
					Ω(results[0].Winner).Should(Equal("REP-2"))

					instancesOn0 := client.SimulatedInstances(repGuids[0])
					instancesOn1 := client.SimulatedInstances(repGuids[1])

					Ω(instancesOn0).Should(HaveLen(50))
					Ω(instancesOn1).Should(HaveLen(31))
				})
			})

			Context("when there are very many duplicate instances out there", func() {
				BeforeEach(func() {
					initialDistributions[0] = generateSimulatedInstancesForProcessGuid(processGuid, 50, 0, 1)
					initialDistributions[0] = append(initialDistributions[0], generateSimulatedInstancesForProcessGuid(processGuid, 90-50, 1, 1)...)

					initialDistributions[1] = generateSimulatedInstancesForProcessGuid(processGuid, 30, 0, 1)
					initialDistributions[1] = append(initialDistributions[1], generateSimulatedInstancesForProcessGuid(processGuid, 90-30, 1, 1)...)

					initialDistributions[2] = generateSimulatedInstancesForProcessGuid(processGuid, 70, 0, 1)
					initialDistributions[2] = append(initialDistributions[2], generateSimulatedInstancesForProcessGuid(processGuid, 90-70, 1, 1)...)
				})

				It("should stop all but 1", func() {
					stopAuctions := []models.LRPStopAuction{
						{
							ProcessGuid: processGuid,
							Index:       1,
						},
					}

					results := auctionDistributor.HoldStopAuctions(stopAuctions, repGuids)
					Ω(results).Should(HaveLen(1))
					Ω(results[0].Winner).Should(Equal("REP-2"))

					instancesOn0 := client.SimulatedInstances(repGuids[0])
					instancesOn1 := client.SimulatedInstances(repGuids[1])
					instancesOn2 := client.SimulatedInstances(repGuids[2])

					Ω(instancesOn0).Should(HaveLen(50))
					Ω(instancesOn1).Should(HaveLen(31))
					Ω(instancesOn2).Should(HaveLen(70))
				})
			})
		})
	})
})
