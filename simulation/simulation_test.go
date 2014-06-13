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

	newSimulatedInstance := func(appGuid string, index int, memoryMB int) auctiontypes.SimulatedInstance {
		return auctiontypes.SimulatedInstance{
			ProcessGuid:  appGuid,
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

	newLRPStartAuction := func(appGuid string, memoryMB int) models.LRPStartAuction {
		return models.LRPStartAuction{
			ProcessGuid:  appGuid,
			InstanceGuid: util.NewGuid("INS"),
			MemoryMB:     memoryMB,
			DiskMB:       1,
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

	generateLRPStartAuctionsForAppGuid := func(numInstances int, appGuid string, memoryMB int) []models.LRPStartAuction {
		instances := []models.LRPStartAuction{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newLRPStartAuction(appGuid, memoryMB))
		}
		return instances
	}

	BeforeEach(func() {
		util.ResetGuids()
		initialDistributions = map[int][]auctiontypes.SimulatedInstance{}
	})

	JustBeforeEach(func() {
		for index, simulatedInstances := range initialDistributions {
			client.SetSimulatedInstances(guids[index], simulatedInstances)
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
				Context("with single-instance and multi-instance apps apps", func() {
					It("should distribute evenly", func() {
						instances := []models.LRPStartAuction{}

						instances = append(instances, generateUniqueLRPStartAuctions(n1apps[i]/2, 1)...)
						instances = append(instances, generateLRPStartAuctionsWithRandomSVGColors(n1apps[i]/2, 1)...)
						instances = append(instances, generateUniqueLRPStartAuctions(n2apps[i]/2, 2)...)
						instances = append(instances, generateLRPStartAuctionsWithRandomSVGColors(n2apps[i]/2, 2)...)
						instances = append(instances, generateUniqueLRPStartAuctions(n4apps[i]/2, 4)...)
						instances = append(instances, generateLRPStartAuctionsWithRandomSVGColors(n4apps[i]/2, 4)...)

						permutedInstances := make([]models.LRPStartAuction, len(instances))
						for i, index := range util.R.Perm(len(instances)) {
							permutedInstances[i] = instances[index]
						}

						report := auctionDistributor.HoldAuctionsFor(instances, guids[:nexec[i]], auctionrunner.DefaultRules)

						visualization.PrintReport(client, report.AuctionResults, guids[:nexec[i]], report.AuctionDuration, auctionrunner.DefaultRules)

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

						report := auctionDistributor.HoldAuctionsFor(instances, guids[:nexec[i]], auctionrunner.DefaultRules)

						visualization.PrintReport(client, report.AuctionResults, guids[:nexec[i]], report.AuctionDuration, auctionrunner.DefaultRules)

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
						instances := generateLRPStartAuctionsForAppGuid(napps[i], "red", 1)

						report := auctionDistributor.HoldAuctionsFor(instances, guids[:nexec[i]], auctionrunner.DefaultRules)

						visualization.PrintReport(client, report.AuctionResults, guids[:nexec[i]], report.AuctionDuration, auctionrunner.DefaultRules)

						svgReport.DrawReportCard(i, 2, report)
						reports = append(reports, report)
					})
				})
			}
		})
	})
})
