package simulation_test

import (
	"time"

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
			DesiredLRP: models.DesiredLRP{
				ProcessGuid: processGuid,
				MemoryMB:    memoryMB,
				DiskMB:      1,
			},

			InstanceGuid: util.NewGuid("INS"),
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

	generateLRPStartAuctionsWithRandomColor := func(numInstances int, memoryMB int, colors []string) []models.LRPStartAuction {
		instances := []models.LRPStartAuction{}
		for i := 0; i < numInstances; i++ {
			color := colors[util.R.Intn(len(colors))]
			instances = append(instances, newLRPStartAuction(color, memoryMB))
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

	runStartAuction := func(startAuctions []models.LRPStartAuction, nCells int, i int, j int) {
		t := time.Now()
		results := auctionDistributor.HoldStartAuctions(nCells, startAuctions, repAddresses[:nCells], auctionrunner.DefaultStartAuctionRules)
		duration := time.Since(t)
		report := &visualization.Report{
			RepAddresses:    repAddresses[:nCells],
			AuctionResults:  results,
			InstancesByRep:  visualization.FetchAndSortInstances(client, repAddresses[:nCells]),
			AuctionDuration: duration,
		}
		visualization.PrintReport(client, len(startAuctions), results, repAddresses[:nCells], duration, auctionrunner.DefaultStartAuctionRules)
		svgReport.DrawReportCard(i, j, report)
		reports = append(reports, report)
	}

	BeforeEach(func() {
		util.ResetGuids()
		initialDistributions = map[int][]auctiontypes.SimulatedInstance{}
	})

	JustBeforeEach(func() {
		for index, simulatedInstances := range initialDistributions {
			client.SetSimulatedInstances(repAddresses[index], simulatedInstances)
		}
	})

	Describe("Experiments", func() {
		Context("Small Cold Starts", func() {
			for repeat := 0; repeat < 4; repeat++ {
				repeat := repeat
				It("should distribute evenly for a very small distribution", func() {
					napps := 8
					nexecutors := 4

					instances := generateUniqueLRPStartAuctions(napps, 1)

					runStartAuction(instances, nexecutors, repeat, 0)
				})

				It("should distribute evenly for a small distribution", func() {
					napps := 40
					nexecutors := 10

					instances := generateUniqueLRPStartAuctions(napps, 1)

					runStartAuction(instances, nexecutors, repeat, 1)
				})
			}
		})

		Context("Large Cold Starts", func() {
			nexecutors := []int{25, 4 * 25}
			n1apps := []int{1800, 4 * 1800}
			n2apps := []int{200, 4 * 200}
			n4apps := []int{50, 4 * 50}
			for i := range nexecutors {
				i := i
				Context("with single-instance and multi-instance apps", func() {
					It("should distribute evenly", func() {
						instances := []models.LRPStartAuction{}
						colors := []string{"purple", "red", "orange", "teal", "gray", "blue", "pink", "green", "lime", "cyan", "lightseagreen", "brown"}

						instances = append(instances, generateUniqueLRPStartAuctions(n1apps[i]/2, 1)...)
						instances = append(instances, generateLRPStartAuctionsWithRandomColor(n1apps[i]/2, 1, colors[:4])...)
						instances = append(instances, generateUniqueLRPStartAuctions(n2apps[i]/2, 2)...)
						instances = append(instances, generateLRPStartAuctionsWithRandomColor(n2apps[i]/2, 2, colors[4:8])...)
						instances = append(instances, generateUniqueLRPStartAuctions(n4apps[i]/2, 4)...)
						instances = append(instances, generateLRPStartAuctionsWithRandomColor(n4apps[i]/2, 4, colors[8:12])...)

						permutedInstances := make([]models.LRPStartAuction, len(instances))
						for i, index := range util.R.Perm(len(instances)) {
							permutedInstances[i] = instances[index]
						}

						runStartAuction(permutedInstances, nexecutors[i], i, 2)
					})
				})
			}
		})

		Context("Imbalanced scenario (e.g. a deploy)", func() {
			nexecutors := []int{100, 100}
			nempty := []int{5, 1}
			napps := []int{500, 100}

			for i := range nexecutors {
				i := i
				Context("scenario", func() {
					BeforeEach(func() {
						for j := 0; j < nexecutors[i]-nempty[i]; j++ {
							initialDistributions[j] = generateUniqueSimulatedInstances(50, 0, 1)
						}
					})

					It("should distribute evenly", func() {
						instances := generateUniqueLRPStartAuctions(napps[i], 1)

						runStartAuction(instances, nexecutors[i], i+2, 2)
					})
				})
			}
		})

		Context("The Watters demo", func() {
			nexecutors := []int{4, 10, 30, 100}
			napps := []int{20, 80, 200, 400}

			for i := range nexecutors {
				i := i

				Context("scenario", func() {
					BeforeEach(func() {
						for j := 0; j < nexecutors[i]; j++ {
							initialDistributions[j] = generateUniqueSimulatedInstances(util.RandomIntIn(78, 80), 0, 1)
						}
					})

					It("should distribute evenly", func() {
						instances := generateLRPStartAuctionsForProcessGuid(napps[i], "red", 1)

						runStartAuction(instances, nexecutors[i], i, 3)
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

					results := auctionDistributor.HoldStopAuctions(numCells, stopAuctions, repAddresses)
					Ω(results).Should(HaveLen(1))
					Ω(results[0].Winner).Should(Equal(repAddresses[1].RepGuid))

					instancesOn0 := client.SimulatedInstances(repAddresses[0])
					instancesOn1 := client.SimulatedInstances(repAddresses[1])

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

					results := auctionDistributor.HoldStopAuctions(numCells, stopAuctions, repAddresses)
					Ω(results).Should(HaveLen(1))
					Ω(results[0].Winner).Should(Equal(repAddresses[0].RepGuid))

					instancesOn0 := client.SimulatedInstances(repAddresses[0])
					instancesOn1 := client.SimulatedInstances(repAddresses[1])

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

					results := auctionDistributor.HoldStopAuctions(numCells, stopAuctions, repAddresses)
					Ω(results).Should(HaveLen(1))
					Ω(results[0].Winner).Should(Equal(repAddresses[1].RepGuid))

					instancesOn0 := client.SimulatedInstances(repAddresses[0])
					instancesOn1 := client.SimulatedInstances(repAddresses[1])

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

					results := auctionDistributor.HoldStopAuctions(numCells, stopAuctions, repAddresses)
					Ω(results).Should(HaveLen(1))
					Ω(results[0].Winner).Should(Equal(repAddresses[1].RepGuid))

					instancesOn0 := client.SimulatedInstances(repAddresses[0])
					instancesOn1 := client.SimulatedInstances(repAddresses[1])
					instancesOn2 := client.SimulatedInstances(repAddresses[2])

					Ω(instancesOn0).Should(HaveLen(50))
					Ω(instancesOn1).Should(HaveLen(31))
					Ω(instancesOn2).Should(HaveLen(70))
				})
			})
		})
	})
})
