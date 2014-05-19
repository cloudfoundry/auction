package visualization

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/GaryBoone/GoStats/stats"
	. "github.com/onsi/gomega"

	"github.com/ajstarks/svgo"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

const border = 5
const instanceSize = 4
const instanceSpacing = 1
const instanceBoxSize = instanceSize*100 + instanceSpacing*99

const headerHeight = 100

const graphWidth = 300
const graphTextX = 50
const graphBinX = 55
const binHeight = 14
const binSpacing = 2
const maxBinLength = graphWidth - graphBinX

const ReportCardWidth = border*3 + instanceBoxSize + graphWidth
const ReportCardHeight = border*3 + instanceBoxSize

type SVGReport struct {
	SVG            *svg.SVG
	f              *os.File
	scores         []float64
	communications []float64
	waitTimes      []float64
	width          int
	height         int
}

func StartSVGReport(path string, width, height int) *SVGReport {
	f, err := os.Create(path)
	Ω(err).ShouldNot(HaveOccurred())
	s := svg.New(f)
	s.Start(width*ReportCardWidth, headerHeight+height*ReportCardHeight)
	return &SVGReport{
		f:      f,
		SVG:    s,
		width:  width,
		height: height,
	}
}

func (r *SVGReport) Done() {
	r.drawResults()
	r.SVG.End()
	r.f.Close()
}

func (r *SVGReport) DrawHeader(communicationMode string, rules auctiontypes.AuctionRules, maxConcurrent int) {
	rulesString := fmt.Sprintf("%#v", rules)
	header := fmt.Sprintf("%s - MaxConcurrent:%d - %s ", communicationMode, maxConcurrent, rulesString[19:len(rulesString)-1])
	r.SVG.Text(border, 40, header, `text-anchor:start;font-size:32px;font-family:Helvetica Neue`)
}

func (r *SVGReport) drawResults() {
	r.SVG.Text(border, 90, fmt.Sprintf("Score: %.2f | Wait Time: %.2fs | Communications: %.0f", stats.StatsSum(r.scores), stats.StatsSum(r.waitTimes), stats.StatsSum(r.communications)), `text-anchor:start;font-size:32px;font-family:Helvetica Neue`)
}

func (r *SVGReport) DrawReportCard(x, y int, report *Report) {
	r.SVG.Translate(x*ReportCardWidth, headerHeight+y*ReportCardHeight)

	r.drawInstances(report)
	y = r.drawDurationsHistogram(report)
	y = r.drawRoundsHistogram(report, y+binSpacing*4)
	r.drawText(report, y+binSpacing*4)

	r.scores = append(r.scores, report.DistributionScore())
	r.communications = append(r.communications, report.CommStats().Total)
	r.waitTimes = append(r.waitTimes, report.AuctionDuration.Seconds())

	r.SVG.Gend()
}

func (r *SVGReport) drawInstances(report *Report) {
	y := border
	for _, guid := range report.RepGuids {
		x := border
		r.SVG.Rect(x, y, instanceBoxSize, instanceSize, "fill:#f7f7f7")
		instances := report.InstancesByRep[guid]
		for _, instance := range instances {
			instanceWidth := int(float64(instanceSize) * instance.Resources.MemoryMB)
			style := instanceStyle(instance.AppGuid)
			if report.IsAuctionedInstance(instance) {
				r.SVG.Rect(x, y, instanceWidth, instanceSize, style)
			} else {
				r.SVG.Rect(x+1, y+1, instanceWidth-2, instanceSize-2, style)
			}
			x += instanceWidth + instanceSpacing
		}
		y += instanceSize + instanceSpacing
	}
}

func (r *SVGReport) drawDurationsHistogram(report *Report) int {
	waitTimes := []float64{}
	for _, result := range report.AuctionResults {
		waitTimes = append(waitTimes, result.Duration.Seconds())
	}
	sort.Sort(sort.Float64Slice(waitTimes))

	bins := binUp([]float64{0, 0.25, 0.5, 1, 2, 5, 10, 20, 40, 1e9}, waitTimes)
	labels := []string{"<0.25s", "0.25-0.5s", "0.5-1s", "1-2s", "2-5s", "5-10s", "10-20s", "20-40s", ">40s"}

	r.SVG.Translate(border*2+instanceBoxSize, border)

	yBottom := r.drawHistogram(bins, labels)

	r.SVG.Gend()

	return yBottom + border //'cause of the translate
}

func (r *SVGReport) drawRoundsHistogram(report *Report, y int) int {
	rounds := []float64{}
	for _, result := range report.AuctionResults {
		rounds = append(rounds, float64(result.NumRounds))
	}
	sort.Sort(sort.Float64Slice(rounds))

	bins := binUp([]float64{0, 1, 2, 3, 4, 5, 10, 20, 40, 1e9}, rounds)
	labels := []string{"1 round", "2 rounds", "3 rounds", "4 rounds", "5 rounds", "5-10", "10-20", "20-40", ">40"}

	r.SVG.Translate(border*2+instanceBoxSize, y)

	yBottom := r.drawHistogram(bins, labels)

	r.SVG.Gend()

	return yBottom + y
}

func (r *SVGReport) drawText(report *Report, y int) {
	commStats := report.CommStats()
	bidStats := report.BiddingTimeStats()
	waitStats := report.WaitTimeStats()

	missing := ""
	missingInstances := report.NMissingInstances()
	if missingInstances > 0 {
		missing = fmt.Sprintf("MISSING %d (%.2f%%)", missingInstances, float64(missingInstances)/float64(report.NAuctions())*100)
	}

	lines := []string{
		fmt.Sprintf("%d over %d Reps %s", report.NAuctions(), report.NReps(), missing),
		fmt.Sprintf("%.2fs (%.2f a/s)", report.AuctionDuration.Seconds(), report.AuctionsPerSecond()),
		fmt.Sprintf("Dist: %.3f => %.3f", report.InitialDistributionScore(), report.DistributionScore()),
		fmt.Sprintf("%.0f Comm | %.1f ± %.1f | %.0f - %.0f", commStats.Total, commStats.Mean, commStats.StdDev, commStats.Min, commStats.Max),
	}
	statLines := []string{
		"Wait Times",
		fmt.Sprintf("...%.2fs | %.2f ± %.2f", report.AuctionDuration.Seconds(), waitStats.Mean, waitStats.StdDev),
		fmt.Sprintf("...%.3f - %.3f", waitStats.Min, waitStats.Max),
		"Bidding Times",
		fmt.Sprintf("...%s | %.2f ± %.2f", time.Duration(bidStats.Total*float64(time.Second)), bidStats.Mean, bidStats.StdDev),
		fmt.Sprintf("...%.3f - %.2f", bidStats.Min, bidStats.Max),
	}

	r.SVG.Translate(border*2+instanceBoxSize, y)
	r.SVG.Gstyle("font-family:Helvetica Neue")
	r.SVG.Textlines(8, 8, lines, 16, 18, "#333", "start")
	r.SVG.Textlines(8, 80, statLines, 13, 16, "#333", "start")
	r.SVG.Gend()
	r.SVG.Gend()
}

func (r *SVGReport) drawHistogram(bins []float64, labels []string) int {
	y := 0
	for i, percentage := range bins {
		r.SVG.Rect(graphBinX, y, maxBinLength, binHeight, `fill:#eee`)
		r.SVG.Text(graphTextX, y+binHeight-4, labels[i], `text-anchor:end;font-size:10px;font-family:Helvetica Neue`)
		if percentage > 0 {
			r.SVG.Rect(graphBinX, y, int(percentage*float64(maxBinLength)), binHeight, `fill:#333`)
			r.SVG.Text(graphBinX+binSpacing, y+binHeight-4, fmt.Sprintf("%.1f%%", percentage*100.0), `text-anchor:start;font-size:10px;font-family:Helvetica Neue;fill:#fff`)
		}
		y += binHeight + binSpacing
	}

	return y
}

func binUp(binBoundaries []float64, sortedData []float64) []float64 {
	bins := make([]float64, len(binBoundaries)-1)
	currentBin := 0
	for _, d := range sortedData {
		for binBoundaries[currentBin+1] < d {
			currentBin += 1
		}
		bins[currentBin] += 1
	}

	for i := range bins {
		bins[i] = (bins[i] / float64(len(sortedData)))
	}

	return bins
}

func instanceStyle(appGuid string) string {
	components := strings.Split(appGuid, "-")
	color := appGuid
	if len(components) > 1 {
		color = components[len(components)-1]
	}
	return "fill:" + color + ";" + "stroke:none"
}
