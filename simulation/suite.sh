set -e -x

ginkgo -- -algorithm=all_rescore -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -algorithm=all_reserve -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -algorithm=reserve_n_best -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -algorithm=pick_best -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -algorithm=random -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -algorithm=pick_among_best -maxBiddingPool=20 -maxConcurrent=20

ginkgo -- -algorithm=all_rescore -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -algorithm=all_reserve -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -algorithm=reserve_n_best -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -algorithm=pick_best -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -algorithm=random -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -algorithm=pick_among_best -maxBiddingPool=100 -maxConcurrent=20

ginkgo -- -algorithm=all_rescore -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -algorithm=all_reserve -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -algorithm=reserve_n_best -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -algorithm=pick_best -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -algorithm=random -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -algorithm=pick_among_best -maxBiddingPool=20 -maxConcurrent=100

ginkgo -- -algorithm=all_rescore -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -algorithm=all_reserve -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -algorithm=reserve_n_best -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -algorithm=pick_best -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -algorithm=random -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -algorithm=pick_among_best -maxBiddingPool=20 -maxConcurrent=1000

ginkgo -- -algorithm=all_rescore -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -algorithm=all_reserve -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -algorithm=reserve_n_best -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -algorithm=pick_best -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -algorithm=random -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -algorithm=pick_among_best -maxBiddingPool=100 -maxConcurrent=100

ginkgo -- -algorithm=all_rescore -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -algorithm=all_reserve -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -algorithm=reserve_n_best -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -algorithm=pick_best -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -algorithm=random -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -algorithm=pick_among_best -maxBiddingPool=100 -maxConcurrent=1000

ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_rescore -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=20 -maxConcurrent=20

ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_rescore -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=100 -maxConcurrent=20

ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_rescore -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=20 -maxConcurrent=100

ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_rescore -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=20 -maxConcurrent=1000

ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_rescore -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=100 -maxConcurrent=100

ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_rescore -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -communicationMode=nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=100 -maxConcurrent=1000
