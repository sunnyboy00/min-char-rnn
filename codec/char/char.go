package char

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/owulveryck/min-char-rnn/rnn"
	"gonum.org/v1/gonum/stat/distuv"
)

type configuration struct {
	Epoch     int    `default:"100" required:"true"`
	Input     string `envconfig:"input_file" default:"" required:"true"`
	VocabFile string `envconfig:"vocab_file" default:"" required:"true"`
	BatchSize int    `envconfig:"BATCH_SIZE" default:"25" required:"true"`
	Choice    string `default:"hard" required:"true"`
}

var conf configuration

const envPrefix = "CHAR_CODEC"

// Configure the codec via environment variables
func Configure() error {

	err := envconfig.Process(envPrefix, &conf)
	if err != nil {
		return err
	}
	if conf.BatchSize == 0 {
		return errors.New("BATCH_SIZE cannot be null")
	}
	if _, err := os.Stat(conf.VocabFile); err != nil {
		return err
	}
	if _, err := os.Stat(conf.Input); err != nil {
		return err
	}
	return nil
}

// Char is the basic codec for feeding a RNN with text
type Char struct {
	loss       float64
	smoothLoss float64
	batchSize  int
	input      io.ReadSeeker
	runesToIx  map[rune]int
	ixToRunes  map[int]rune
}

// NewChar ...
func NewChar() (*Char, error) {
	err := Configure()
	if err != nil {
		return nil, err
	}
	runesToIx, ixToRunes, err := getVocabIndexesFromFile(conf.VocabFile)
	if err != nil {
		return nil, err
	}
	r, err := os.Open(conf.Input)
	if err != nil {
		return nil, err
	}
	return &Char{
		loss:       0,
		batchSize:  conf.BatchSize,
		ixToRunes:  ixToRunes,
		runesToIx:  runesToIx,
		smoothLoss: -math.Log(float64(1)/float64(len(runesToIx))) * float64(conf.BatchSize),
		input:      r,
	}, nil
}

// Decode an array of inputs and returns an io.Reader
// the input is an array of 1-of-K encoded vectors
func (c *Char) Decode(xs [][]float64) io.Reader {
	var output bytes.Buffer
	buf := bufio.NewWriter(&output)
	for _, x := range xs {
		// Find the index of 1
		idx := 0
		for idx = range x {
			if x[idx] == 1 {
				break
			}
		}
		_, err := buf.WriteRune(c.ixToRunes[idx])
		fmt.Printf("%c", c.ixToRunes[idx])
		if err != nil {
			log.Println(err)
		}
	}
	err := buf.Flush()
	if err != nil {
		log.Println(err)
	}
	return &output
}

// Encode the io.Reader into an slice composed of
// 1-of-K encoded vectors
func (c *Char) Encode(r io.Reader) [][]float64 {
	rdr := bufio.NewReader(r)
	var xs [][]float64
	for {
		if char, _, err := rdr.ReadRune(); err != nil {
			if err == io.EOF {
				// Restart the training if it's not the last epoch
				break
			}
			log.Fatal(err)
		} else {
			oneOfK := make([]float64, len(c.runesToIx))
			oneOfK[c.runesToIx[char]] = 1
			xs = append(xs, oneOfK)
		}
	}
	return xs
}

// Feed returns a channel that will be filled with TrainingSets
// its triggers a go-routine that reads the input and
// that is putting some data in the channel
func (c *Char) Feed() <-chan rnn.TrainingSet {
	feed := make(chan rnn.TrainingSet, 1)
	go func(feed chan<- rnn.TrainingSet) {
		r := bufio.NewReader(c.input)
		tset := rnn.TrainingSet{
			Inputs:  make([][]float64, conf.BatchSize),
			Targets: make([][]float64, conf.BatchSize),
		}
		for epoch := 0; epoch < conf.Epoch; epoch++ {
			if _, err := c.input.Seek(0, io.SeekStart); err != nil {
				log.Fatal(err)
			}
			for {
				var char rune
				var err error
				for i := 0; i < conf.BatchSize+1; i++ {
					if char, _, err = r.ReadRune(); err != nil {
						if err == io.EOF {
							break
						}
						log.Fatal(err)
					}
					oneOfK := make([]float64, len(c.runesToIx))
					oneOfK[c.runesToIx[char]] = 1

					switch i {
					case 0:
						tset.Inputs[i] = oneOfK
					case conf.BatchSize:
						tset.Targets[i-1] = oneOfK
					default:
						tset.Inputs[i] = oneOfK
						tset.Targets[i-1] = oneOfK
					}
				}
				if err == io.EOF {
					break
				}
				feed <- rnn.CopyOf(tset)
			}
		}
		close(feed)
	}(feed)
	return feed
}

// NewRNN returns a neural network suitable for this codec
func (c *Char) NewRNN() *rnn.RNN {
	return rnn.NewRNN(len(c.runesToIx), len(c.ixToRunes))
}

// ApplyDist applies  a distribution according to the configuration of the neural network
func (c *Char) ApplyDist(p []float64) []float64 {
	output := make([]float64, len(p))
	switch conf.Choice {
	case "soft":
		sample := distuv.NewCategorical(p, rand.New(rand.NewSource(time.Now().UnixNano())))
		output[int(sample.Rand())] = 1
	default:
		best := float64(0)
		bestIdx := 0
		for i, v := range p {
			if v > best {
				best = v
				bestIdx = i
			}
		}
		output[bestIdx] = 1
	}

	return output
}

// SetLoss sets the loss and the smoothLoss
func (c *Char) SetLoss(loss float64) {
	c.loss = loss
	c.smoothLoss = c.smoothLoss*0.999 + loss*0.001
}

// Infos ...
type Infos struct {
	SmoothLoss float64
}

// MarshalJSON ...
func (i Infos) MarshalJSON() ([]byte, error) {
	return json.Marshal(i)
}

// GetInfos ...
func (c *Char) GetInfos() json.Marshaler {
	return Infos{
		c.smoothLoss,
	}
}

// GobEncode ...
func (c *Char) GobEncode() ([]byte, error) {
	return nil, nil
}

// GobDecode ...
func (c *Char) GobDecode([]byte) error {
	return nil
}
