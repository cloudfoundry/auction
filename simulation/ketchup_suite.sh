set -e -x
mkdir -p ./runs

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_rebid -maxBiddingPool=0.2 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=0.2 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=0.2 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=0.2 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=0.2 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=0.2 -maxConcurrent=20

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_rebid -maxBiddingPool=1.0 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=1.0 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=1.0 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=1.0 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=1.0 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=1.0 -maxConcurrent=20

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_rebid -maxBiddingPool=0.2 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=0.2 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=0.2 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=0.2 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=0.2 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=0.2 -maxConcurrent=100

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_rebid -maxBiddingPool=0.2 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=0.2 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=0.2 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=0.2 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=0.2 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=0.2 -maxConcurrent=1000

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_rebid -maxBiddingPool=1.0 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=1.0 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=1.0 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=1.0 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=1.0 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=1.0 -maxConcurrent=100

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_rebid -maxBiddingPool=1.0 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=1.0 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=1.0 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=1.0 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=1.0 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=1.0 -maxConcurrent=1000
