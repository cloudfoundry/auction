package auctionrunner

import "github.com/cloudfoundry-incubator/rep"

type Cell struct {
	Guid   string
	client rep.Client
	state  rep.CellState

	workToCommit rep.Work
}

func NewCell(guid string, client rep.Client, state rep.CellState) *Cell {
	return &Cell{
		Guid:   guid,
		client: client,
		state:  state,
	}
}

func (c *Cell) MatchRootFS(rootFS string) bool {
	return c.state.MatchRootFS(rootFS)
}

func (c *Cell) ScoreForLRP(lrp *rep.LRP) (float64, error) {
	err := c.state.ResourceMatch(&lrp.Resource)
	if err != nil {
		return 0, err
	}

	var numberOfInstancesWithMatchingProcessGuid float64 = 0
	for i := range c.state.LRPs {
		if c.state.LRPs[i].ProcessGuid == lrp.ProcessGuid {
			numberOfInstancesWithMatchingProcessGuid++
		}
	}

	resourceScore := c.state.ComputeScore(&lrp.Resource) + numberOfInstancesWithMatchingProcessGuid
	return resourceScore, nil
}

func (c *Cell) ScoreForTask(task *rep.Task) (float64, error) {
	err := c.state.ResourceMatch(&task.Resource)
	if err != nil {
		return 0, err
	}

	return c.state.ComputeScore(&task.Resource), nil
}

func (c *Cell) ReserveLRP(lrp *rep.LRP) error {
	err := c.state.ResourceMatch(&lrp.Resource)
	if err != nil {
		return err
	}

	c.state.AddLRP(lrp)
	c.workToCommit.LRPs = append(c.workToCommit.LRPs, *lrp)
	return nil
}

func (c *Cell) ReserveTask(task *rep.Task) error {
	err := c.state.ResourceMatch(&task.Resource)
	if err != nil {
		return err
	}

	c.state.AddTask(task)
	c.workToCommit.Tasks = append(c.workToCommit.Tasks, *task)
	return nil
}

func (c *Cell) Commit() rep.Work {
	if len(c.workToCommit.LRPs) == 0 && len(c.workToCommit.Tasks) == 0 {
		return rep.Work{}
	}

	failedWork, err := c.client.Perform(c.workToCommit)
	if err != nil {
		//an error may indicate partial failure
		//in this case we don't reschedule work in order to make sure we don't
		//create duplicates of things -- we'll let the converger figure things out for us later
		return rep.Work{}
	}
	return failedWork
}
