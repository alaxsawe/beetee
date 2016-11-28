package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"io"
	//"time"
)

const (
	ChokeMsg = iota
	UnchokeMsg
	InterestedMsg
	NotInterestedMsg
	HaveMsg
	BitFieldMsg
	RequestMsg
	BlockMsg // rather than PieceMsg
	CancelMsg
	PortMsg
)

/*###################################################
Recieving Messages
######################################################*/

func (p *Peer) decodePieceMessage(msg []byte) {
	if len(msg[8:]) < 1 {
		return
	}
	index := binary.BigEndian.Uint32(msg[:4])
	begin := binary.BigEndian.Uint32(msg[4:8])
	data := msg[8:]
	// Blocks...
	block := &Block{index: index, offset: begin, data: data}
	Pieces[index].chanBlocks <- block
	if len(Pieces[index].chanBlocks) == cap(Pieces[index].chanBlocks) {
		Pieces[index].writeBlocks()
		//Pieces[index].success <- true
	}
}

func (p *Piece) writeBlocks() {
	//p.pending.Done() // not waiting on any more blocks
	if len(p.chanBlocks) < cap(p.chanBlocks) {
		logger.Printf("The block channel for %d is not full", p.index)
		return
	}
	for {
		b := <-p.chanBlocks // NOTE: b for block
		copy(p.data[int(b.offset):int(b.offset)+blocksize],
			b.data)
		if len(p.chanBlocks) < 1 {
			break
		}
	}
	if p.hash != sha1.Sum(p.data) {
		p.data = nil
		p.data = make([]byte, p.size)
		logger.Printf("Unable to Write Blocks to Piece %d",
			p.index)
		return
	}
	p.verified = true
	logger.Printf("Piece at %d is successfully written", p.index)
	ioChan <- p
	p.success <- true
}

// 19 bytes
func (p *Peer) decodeHaveMessage(msg []byte) {
	index := binary.BigEndian.Uint32(msg)
	p.bitfield[index] = true
}

// NOTE: The bitfield will be sent with padding if the size is
// not divisible by eight.
// Thank you Tulva RC bittorent client for this algorithm
// github.com/jtakkala/tulva/
func (p *Peer) decodeBitfieldMessage(bitfield []byte) {
	p.bitfield = make([]bool, len(Pieces))
	// For each byte, look at the bits
	// NOTE: that is 8 * 8
	for i := 0; i < len(p.bitfield); i++ {
		for j := 0; j < 8; j++ {
			index := i*8 + j
			if index >= len(Pieces) {
				break // Hit padding bits
			}

			byte := bitfield[i]              // Within bytes
			bit := (byte >> uint32(7-j)) & 1 // some shifting
			p.bitfield[index] = bit == 1     // if bit is true
		}
	}
}

func (p *Peer) decodeRequestMessage(msg []byte) {
}

func (p *Peer) decodeCancelMessage(msg []byte) {
}

func (p *Peer) decodePortMessage(msg []byte) {
}

// sendHandShake asks another client to accept your connection.
func (p *Peer) decodeHandShake(shake []byte) error {
	///<pstrlen><pstr><reserved><info_hash><peer_id>
	// 68 bytes long.
	pstrlen := byte(19) // or len(pstr)
	pstr := []byte{'B', 'i', 't', 'T', 'o', 'r',
		'r', 'e', 'n', 't', ' ', 'p', 'r',
		'o', 't', 'o', 'c', 'o', 'l'}
	reserved := make([]byte, 8)
	info := Torrent.InfoHash[:]
	id := PeerId[:] // my peerId NOTE: Global

	// TODO: Check for Length
	if !bytes.Equal(shake[1:20], pstr) {
		return errors.New("Protocol does not match")
	}
	if !bytes.Equal(shake[28:48], info) {
		return errors.New("InfoHash Does not match")
	}
	p.id = string(shake[48:68])

	var n int
	var err error
	writer := bufio.NewWriter(p.conn)
	// Handshake message:
	// Send handshake message
	err = writer.WriteByte(pstrlen)
	if err != nil {
		return err
	}
	n, err = writer.Write(pstr)
	if err != nil || n != len(pstr) {
		return err
	}
	n, err = writer.Write(reserved)
	if err != nil || n != len(reserved) {
		return err
	}
	n, err = writer.Write(info)
	if err != nil || n != len(info) {
		return err
	}
	n, err = writer.Write(id)
	if err != nil || n != len(id) {
		return err
	}
	err = writer.Flush() // TODO: Do I need to Flush?
	if err != nil {
		return err
	}
	// receive confirmation
	return nil
}

/*###################################################
Sending Messages
######################################################*/

// sendHandShake asks another client to accept your connection.
func (p *Peer) sendHandShake() error {
	///<pstrlen><pstr><reserved><info_hash><peer_id>
	// 68 bytes long.
	var n int
	var err error
	writer := bufio.NewWriter(p.conn)
	// Handshake message:
	pstrlen := byte(19) // or len(pstr)
	pstr := []byte{'B', 'i', 't', 'T', 'o', 'r',
		'r', 'e', 'n', 't', ' ', 'p', 'r',
		'o', 't', 'o', 'c', 'o', 'l'}
	reserved := make([]byte, 8)
	info := Torrent.InfoHash[:]
	id := PeerId[:] // my peerId NOTE: Global
	// Send handshake message
	err = writer.WriteByte(pstrlen)
	if err != nil {
		return err
	}
	n, err = writer.Write(pstr)
	if err != nil || n != len(pstr) {
		return err
	}
	n, err = writer.Write(reserved)
	if err != nil || n != len(reserved) {
		return err
	}
	n, err = writer.Write(info)
	if err != nil || n != len(info) {
		return err
	}
	n, err = writer.Write(id)
	if err != nil || n != len(id) {
		return err
	}
	err = writer.Flush() // TODO: Do I need to Flush?
	if err != nil {
		return err
	}
	// receive confirmation

	// The response handshake
	shake := make([]byte, 68)
	// Deadlines are set forever
	// // https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
	// err = p.conn.SetDeadline(time.Now().Add(time.Second * 30))
	// if err != nil {
	//	return err
	// }
	n, err = io.ReadFull(p.conn, shake)
	if err != nil {
		return err
	}
	// TODO: Check for Length
	if !bytes.Equal(shake[1:20], pstr) {
		return errors.New("Protocol does not match")
	}
	if !bytes.Equal(shake[28:48], info) {
		return errors.New("InfoHash Does not match")
	}
	p.id = string(shake[48:68])
	return nil
}

// sendStatusMessage sends the status message to peer.
// If sent -1 then a Keep alive message is sent.
func (p *Peer) sendStatusMessage(msg int) error {
	logger.Printf("Sending Status Message: %d to %s", msg, p.id)
	var err error
	buf := make([]byte, 4)
	writer := bufio.NewWriter(p.conn)
	if msg == -1 { // keep alive, do nothing TODO: add ot iota
		binary.BigEndian.PutUint32(buf, 0)
	} else {
		binary.BigEndian.PutUint32(buf, 1)
	}
	writer.Write(buf)
	if err != nil {
		return err
	}
	switch msg { //<len=0001><id=0>
	case ChokeMsg:
		err = writer.WriteByte((uint8)(0))
	case UnchokeMsg:
		err = writer.WriteByte((uint8)(1))
	case InterestedMsg:
		err = writer.WriteByte(byte(2))
	case NotInterestedMsg:
		err = writer.WriteByte((uint8)(3))
	}
	if err != nil {
		return err
	}
	writer.Flush()
	return nil
}

// sendRequestMessage pass in the index of the piece your looking for,
// and the offset of the piece (it's offset index * BLOCKSIZE
func (p *Peer) sendRequestMessage(idx uint32, offset int) error {
	//4-byte message length,1-byte message ID, and payload:
	// <len=0013><id=6><index><begin><length>
	// NOTE: being offset the offset by byte:
	// that is  0, 16K, 13K, etc
	var err error
	writer := bufio.NewWriter(p.conn)
	len := make([]byte, 4)
	binary.BigEndian.PutUint32(len, 13)
	id := byte(RequestMsg)
	// payload
	index := make([]byte, 4)
	binary.BigEndian.PutUint32(index, idx)
	begin := make([]byte, 4)
	binary.BigEndian.PutUint32(begin, uint32(offset))
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(blocksize))
	_, err = writer.Write(len)
	if err != nil {
		return err
	}
	err = writer.WriteByte(id)
	if err != nil {
		return err
	}
	_, err = writer.Write(index)
	if err != nil {
		return err
	}
	_, err = writer.Write(begin)
	if err != nil {
		return err
	}
	_, err = writer.Write(length)
	if err != nil {
		return err
	}
	writer.Flush()
	return nil
}

// FOR TESTING NOTE
func (p *Peer) requestAllPieces() {
	total := len(Pieces)
	//completionSync.Add(total - 1)
	debugger.Printf("Requesting all %d pieces", total)
	for i := 0; i < total; i++ {
		p.requestPiece(i)
	}
}

func (p *Peer) requestPiece(piece int) {
	logger.Printf("Requesting piece %d from peer %s", piece, p.id)
	blocksPerPiece := int(Torrent.Info.PieceLength) / blocksize
	for offset := 0; offset < blocksPerPiece; offset++ {
		err := p.sendRequestMessage(uint32(piece), offset*blocksize)
		if err != nil {
			debugger.Println("Error Requesting", err)
		}
	}
}
