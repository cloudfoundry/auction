package simulation_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/simulation/util"
	"github.com/cloudfoundry-incubator/auction/simulation/visualization"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Ω

var _ = Describe("Auction", func() {
	var initialDistributions map[int][]auctiontypes.LRP

	newLRP := func(processGuid string, index int, memoryMB int) auctiontypes.LRP {
		return auctiontypes.LRP{
			ProcessGuid:  processGuid,
			InstanceGuid: util.NewGuid("INS"),
			Index:        index,
			MemoryMB:     memoryMB,
			DiskMB:       1,
		}
	}

	generateUniqueLRPs := func(numInstances int, index int, memoryMB int) []auctiontypes.LRP {
		instances := []auctiontypes.LRP{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newLRP(util.NewGrayscaleGuid("AAA"), index, memoryMB))
		}
		return instances
	}

	newLRPStartAuction := func(processGuid string, memoryMB int) models.LRPStartAuction {
		return models.LRPStartAuction{
			DesiredLRP: models.DesiredLRP{
				ProcessGuid: processGuid,
				MemoryMB:    memoryMB,
				DiskMB:      1,
				Stack:       "lucid64",
				Domain:      "domain",
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

	workForInstances := func(lrps []auctiontypes.LRP) auctiontypes.Work {
		work := auctiontypes.Work{}
		for _, lrp := range lrps {
			work.LRPStarts = append(work.LRPStarts, models.LRPStartAuction{
				DesiredLRP: models.DesiredLRP{
					ProcessGuid: lrp.ProcessGuid,
					MemoryMB:    lrp.MemoryMB,
					DiskMB:      lrp.DiskMB,
					Stack:       "lucid64",
					Domain:      "domain",
				},

				InstanceGuid: lrp.InstanceGuid,
				Index:        lrp.Index,
			})
		}
		return work
	}

	runStartAuction := func(lrpStartAuctions []models.LRPStartAuction, numCells int, i int, j int) {
		t := time.Now()
		auctionRunnerDelegate.SetCellLimit(numCells)
		for _, startAuction := range lrpStartAuctions {
			auctionRunner.AddLRPStartAuction(startAuction)
		}

		Eventually(auctionRunnerDelegate.ResultSize, time.Minute, 100*time.Millisecond).Should(Equal(len(lrpStartAuctions)))
		duration := time.Since(t)

		cells, _ := auctionRunnerDelegate.FetchCellReps()
		report := visualization.NewReport(len(lrpStartAuctions), cells, auctionRunnerDelegate.Results(), duration)

		visualization.PrintReport(report)
		svgReport.DrawReportCard(i, j, report)
		reports = append(reports, report)
		fmt.Println("Done...")
	}

	BeforeEach(func() {
		util.ResetGuids()
		initialDistributions = map[int][]auctiontypes.LRP{}
	})

	JustBeforeEach(func() {
		wg := &sync.WaitGroup{}
		wg.Add(len(initialDistributions))
		for index, instances := range initialDistributions {
			guid := cellGuid(index)
			instances := instances
			auctionWorkPool.Submit(func() {
				cells[guid].Perform(workForInstances(instances))
				wg.Done()
			})
		}
		wg.Wait()
	})

	Describe("Experiments", func() {
		Context("Small Cold LRPStarts", func() {
			napps := []int{8, 40, 200, 800}
			ncells := []int{4, 10, 20, 40}
			for i := range ncells {
				i := i
				It("should distribute evenly", func() {
					instances := generateUniqueLRPStartAuctions(napps[i], 1)

					runStartAuction(instances, ncells[i], i, 0)
				})
			}
		})

		Context("Large Cold LRPStarts", func() {
			ncells := []int{25, 4 * 25}
			n1apps := []int{1800, 4 * 1800}
			n2apps := []int{200, 4 * 200}
			n4apps := []int{50, 4 * 50}
			for i := range ncells {
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

						runStartAuction(permutedInstances, ncells[i], i, 1)
					})
				})
			}
		})

		Context("Imbalanced scenario (e.g. a deploy)", func() {
			ncells := []int{100, 100}
			nempty := []int{5, 1}
			napps := []int{500, 100}

			for i := range ncells {
				i := i
				Context("scenario", func() {
					BeforeEach(func() {
						for j := 0; j < ncells[i]-nempty[i]; j++ {
							initialDistributions[j] = generateUniqueLRPs(50, 0, 1)
						}
					})

					It("should distribute evenly", func() {
						instances := generateUniqueLRPStartAuctions(napps[i], 1)

						runStartAuction(instances, ncells[i], i+2, 1)
					})
				})
			}
		})

		Context("The Watters demo", func() {
			ncells := []int{4, 10, 30, 100}
			napps := []int{20, 80, 200, 400}

			for i := range ncells {
				i := i

				Context("scenario", func() {
					BeforeEach(func() {
						for j := 0; j < ncells[i]; j++ {
							initialDistributions[j] = generateUniqueLRPs(util.RandomIntIn(78, 80), 0, 1)
						}
					})

					It("should distribute evenly", func() {
						instances := generateLRPStartAuctionsForProcessGuid(napps[i], "red", 1)

						runStartAuction(instances, ncells[i], i, 2)
					})
				})
			}
		})
	})
})
