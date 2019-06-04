package main

import (
	"bytes"
	"os"
	"os/exec"
	"os/signal"

	"github.com/advanderveer/27067dd17/onl/agent"
)

func drawPNG(a *agent.Agent, name string) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	buf := bytes.NewBuffer(nil)
	err = a.Draw(buf)
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("dot", "-Tpng")
	cmd.Stdin = buf
	cmd.Stdout = f
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}

func main() {
	sigs := make(chan os.Signal)
	signal.Notify(sigs)

	cfg1 := agent.DefaultConf()
	cfg2 := agent.DefaultConf()
	cfg1.StartWithStake(1, cfg1.Identity, cfg2.Identity)
	cfg2.StartWithStake(1, cfg1.Identity, cfg2.Identity)

	a1, err := agent.New(cfg1)
	if err != nil {
		panic(err)
	}

	a2, err := agent.New(cfg2)
	if err != nil {
		panic(err)
	}

	err = a1.Join(a2.Addr())
	if err != nil {
		panic(err)
	}

	err = a2.Join(a1.Addr())
	if err != nil {
		panic(err)
	}

	<-sigs

	err = a1.Close()
	if err != nil {
		panic(err)
	}

	err = a2.Close()
	if err != nil {
		panic(err)
	}

	drawPNG(a1, "a1.png")
	drawPNG(a2, "a2.png")
}
