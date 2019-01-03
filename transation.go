package main

import (
	"fmt"
	"strings"
	"bytes"
	"encoding/gob"
	"log"
	"crypto/sha256"
	"encoding/hex"
)

const subsidy = 100
type Transation struct{
	ID []byte
	Vin []TXInput
	Vout []TXOutput

}


type TXInput struct {
	TXid []byte
	Voutindex int
	Signature []byte
	Pubkey  []byte   //公钥
}

type TXOutput struct {
	Value int
	PubkeyHash []byte  //公钥的hash
}

func (out *TXOutput) Lock(address []byte){
	decodeAddress := Base58Decode(address)
	pubkeyhash := decodeAddress[1:len(decodeAddress)-4]
	out.PubkeyHash =  pubkeyhash
}




func (tx Transation) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))

	for i, input := range tx.Vin {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.TXid))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Voutindex))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubkeyHash))
	}

	return strings.Join(lines, "\n")
}


//序列化
func (tx Transation) Serialize() []byte{
	var encoded bytes.Buffer
	enc:= gob.NewEncoder(&encoded)

	err:= enc.Encode(tx)

	if err!=nil{
		log.Panic(err)
	}
	return encoded.Bytes()
}
//计算交易的hash值
func (tx *Transation) Hash() []byte{

	txcopy := *tx
	txcopy.ID = []byte{}

	hash:= sha256.Sum256(txcopy.Serialize())

	return hash[:]
}

//根据金额与地址新建一个输出
func NewTXOutput(value int,address string) * TXOutput{
	txo := &TXOutput{value,nil}
	//txo.PubkeyHash = []byte(address)
	txo.Lock([]byte(address))
	return txo
}



//第一笔coinbase交易
func NewCoinbaseTX(to,data string) *Transation{
	txin := TXInput{[]byte{},-1,nil,[]byte(data)}
	txout := NewTXOutput(subsidy,to)
	tx:= Transation{nil,[]TXInput{txin},[]TXOutput{*txout}}

	tx.ID = tx.Hash()

	return &tx
}

func (out *TXOutput) CanBeUnlockedWith(pubkeyhash []byte) bool{

	return bytes.Compare(out.PubkeyHash,pubkeyhash)==0
}

func (in * TXInput) canUnlockOutputWith(unlockdata []byte) bool{
		lockinghash := HashPubkey(in.Pubkey)

		return bytes.Compare(lockinghash,unlockdata)==0

}

func (tx Transation) IsCoinBase() bool{
	return len(tx.Vin) == 1 && len(tx.Vin[0].TXid) ==0 &&  tx.Vin[0].Voutindex == -1
}


func NewUTXOTransation(from,to string,amount int, bc * Blockchain) *Transation{
		var inputs []TXInput
		var outputs []TXOutput


		wallets,err:= NewWallets()
		if err !=nil{
			log.Panic(err)

		}
		wallet := wallets.GetWallet(from)
		acc,validoutputs := bc.FindSpendableOutputs(HashPubkey(wallet.Publickey),amount)

		if acc < amount{
			log.Panic("Error:Not enough funds")
		}

		for txid,outs := range validoutputs{
			txID ,err := hex.DecodeString(txid)
			if err !=nil{
				log.Panic(err)
			}

			for  _,out := range outs{

				input := TXInput{txID,out,nil,wallet.Publickey}
				inputs  = append(inputs,input)
			}

		}
		outputs  = append(outputs,*NewTXOutput(amount,to))


		if acc > amount{
			outputs = append(outputs,*NewTXOutput(acc-amount,from))
		}


		tx:= Transation{nil,inputs,outputs}
		tx.ID = tx.Hash()

		bc.SignTransation(&tx,wallet.PrivateKey)
		return &tx
}
