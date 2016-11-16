package main

// Flood is when the client run
func Flood() {
	completionSync.Add(len(Pieces) - 1)
	go FillQueue()
	group := Peers[1]
	go group.AskForData()
	//for _, p := range group {
	//n	go p.AskForData()
	//}

}

func (p *Peer) AskForData() {

	err := p.ListenToPeer()
	if err != nil {
		debugger.Println("Error connection", err)
	}
	p.sendStatusMessage(InterestedMsg)
	p.ChokeWg.Wait()

	for {
		if !p.Alive {
			break
		}
		piece := <-PieceQueue
		//debugger.Println(piece.hash, piece.index)
		p.requestPiece(piece.index)
		piece.Pending.Wait()

	}
}

// FillQueue fills the channel for asking for pieces.
func FillQueue() {
	order := DecidePieceOrder()
	for _, val := range order {
		PieceQueue <- Pieces[val]
	}
}

// TODO:
func FindPeerForPiece(idx int) *Peer {
	// TODO: find in alives who has idx m.alives
	for _, peer := range Peers {
		if peer.Alive {
			return peer
		}
	}
	return nil
}

// DecidePieceOrder should return a list of indexes
// of pieces, according to the rarest first
func DecidePieceOrder() []int {
	order := make([]int, 0, len(Pieces))
	for i := 0; i < len(Pieces); i++ {
		if Pieces[i].status == 0 {
			order = append(order, i)
		}
	}
	return order
}
