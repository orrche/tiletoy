package main

import (
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/disintegration/imaging"
	flags "github.com/jessevdk/go-flags"
	yaml "gopkg.in/yaml.v2"
)

// Tile holdes information about a tile, configurations it is and how its edges looks
type Tile struct {
	Filename  string `yaml:"filename"`
	image     image.Image
	Edges     []int `yaml:"edges"`
	Rotations []int `yaml:"rotations"`
	rotation  int
	Weight    float64 `yaml:"weight"`
}

// Config holdes the configurationfile info
type Config struct {
	Tiles []Tile `yaml:"tiles"`
}

var opts struct {
	ConfigFile string `short:"c" long:"config" description:"Configfile to use to the generation" value-name:"config.yml"`
}

// Map keeps trak of the map of different tiles
type Map struct {
	tileMap []Tile
	width   int
	height  int
}

type side int

const (
	up    side = 0
	right side = 1
	down  side = 2
	left  side = 3
)

func getEdge(s side, tile Tile) int {
	return tile.Edges[(int(s)+tile.rotation)%4]
}

func getTile(m Map, x int, y int) *Tile {
	empty := Tile{}
	empty.Edges = []int{0, 0, 0, 0}
	if x < 0 || x >= m.width {
		return &empty
	}

	if y < 0 || y >= m.height {
		return &empty
	}

	t := &m.tileMap[x+y*m.width]
	if t.Filename == "" {
		return nil
	}
	return t
}

func possibleTiles(m Map, tiles []Tile, x int, y int) []Tile {
	res := make([]Tile, 0, 0)

	upTile := getTile(m, x, y-1)
	leftTile := getTile(m, x-1, y)
	downTile := getTile(m, x, y+1)
	rightTile := getTile(m, x+1, y)

	for _, tile := range tiles {
		for _, r := range tile.Rotations {
			tile.rotation = r
			if upTile != nil && getEdge(up, tile) != getEdge(down, *upTile) {
				continue
			}
			if downTile != nil && getEdge(down, tile) != getEdge(up, *downTile) {
				continue
			}
			if leftTile != nil && getEdge(left, tile) != getEdge(right, *leftTile) {
				continue
			}
			if rightTile != nil && getEdge(right, tile) != getEdge(left, *rightTile) {
				continue
			}

			res = append(res, tile)
		}
	}

	return res
}

func main() {
	_, err := flags.ParseArgs(&opts, os.Args)

	if err != nil {
		return
	}
	if opts.ConfigFile == "" {
		opts.ConfigFile = "config.yml"
	}
	rand.Seed(time.Now().UTC().UnixNano())
	m := Map{}
	tileSize := 15
	m.width = 40
	m.height = 40
	m.tileMap = make([]Tile, m.width*m.height, m.width*m.height)

	var config Config
	f, err := os.Open(opts.ConfigFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	for i, tile := range config.Tiles {
		t2f, err := os.Open(tile.Filename)
		if err != nil {
			panic(err)
		}

		config.Tiles[i].image, err = png.Decode(t2f)
		if err != nil {
			panic(err)
		}
		if config.Tiles[i].Weight <= 0 {
			config.Tiles[i].Weight = 1
		}
		if len(config.Tiles[i].Rotations) == 0 {
			config.Tiles[i].Rotations = []int{0, 1, 2, 3}
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, m.width*tileSize, m.height*tileSize))
	for x := 0; x < m.width; x++ {
		for y := 0; y < m.height; y++ {
			tiles := possibleTiles(m, config.Tiles, x, y)

			totWeight := 0.0
			for _, t := range tiles {
				totWeight += t.Weight
			}
			selected := rand.Float64() * totWeight
			for i, t := range tiles {
				selected -= t.Weight
				if selected < 0 {
					m.tileMap[x+y*m.width] = tiles[i]
					break
				}
			}
		}
	}

	for x := 0; x < m.width; x++ {
		for y := 0; y < m.height; y++ {
			tile := getTile(m, x, y)
			var dstImage image.Image
			switch tile.rotation {
			case 0:
				dstImage = tile.image
			case 1:
				dstImage = imaging.Rotate90(tile.image)
			case 2:
				dstImage = imaging.Rotate180(tile.image)
			case 3:
				dstImage = imaging.Rotate270(tile.image)
			}
			draw.Draw(img, image.Rect(x*tileSize+0, y*tileSize+0, x*tileSize+tileSize, y*tileSize+tileSize), dstImage, image.Point{0, 0}, draw.Src)
		}
	}

	f, err = os.Create("output.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	png.Encode(f, img)
}
