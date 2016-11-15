package main

const BLOCKSIZE int = 16384

type Piece struct {
	index      int // redundant
	data       []byte
	numBlocks  int
	blocks     []*Block
	chanBlocks chan *Block
	peer       *Peer
	hash       [20]byte
	size       int64
	have       bool
	length     int
}

type Block struct {
	piece      *Piece // Not necessary?
	offset     int
	length     int  // Not necessary?
	downloaded bool // Not necessary?
	data       []byte
}

// parsePieces parses the big wacky string of sha-1 hashes int
// the Info list of
func (info *TorrentInfo) parsePieces() {
	info.cleanPieces()
	// TODO: set this dynamically
	numBlocks := info.PieceLength / int64(BLOCKSIZE)
	info.BlocksPerPiece = int(numBlocks)
	len := len(info.Pieces)
	info.PieceList = make([]*Piece, 0, len/20)
	for i := 0; i < len; i = i + 20 {
		j := i + 20
		piece := Piece{size: info.PieceLength, numBlocks: int(numBlocks)}
		piece.chanBlocks = make(chan *Block)
		piece.blocks = make([]*Block, 0, piece.numBlocks)
		// Copy to next 20 into Piece Hash
		copy(piece.hash[:], info.Pieces[i:j])
		piece.length = int(info.PieceLength)
		info.PieceList = append(info.PieceList, &piece)
	}
}
