package algo

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func RunVirtualFlow(
	image string,
	program string, ligand []byte, receptor []byte,
	centerX float64, centerY float64, centerZ float64,
	sizeX float64, sizeY float64, sizeZ float64,
	onProgress func(float64)) ([]byte, error) {
	ctx := context.Background()

	dir, err := ioutil.TempDir("", "dah_vf_tmp_*")
	if err != nil {
		return nil, err
	}
	defer func() {
		err = os.RemoveAll(dir)
	}()

	ligandFile := filepath.Join(dir, "ligand.pdbqt")
	err = os.WriteFile(ligandFile, ligand, 0644)
	if err != nil {
		return nil, err
	}

	receptorFile := filepath.Join(dir, "receptor.pdbqt")
	err = os.WriteFile(receptorFile, receptor, 0644)
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(dir, "config.txt")
	err = os.WriteFile(configFile, []byte(fmt.Sprintf(""+
		"ligand = ./data/ligand.pdbqt\n"+
		"receptor = ./data/receptor.pdbqt\n"+
		"out = ./data/output.pdbqt\n"+
		"cpu = %d\n"+
		"exhaustiveness = %d\n"+
		"center_x = %f\n"+
		"center_y = %f\n"+
		"center_z = %f\n"+
		"size_x = %f\n"+
		"size_y = %f\n"+
		"size_z = %f\n",
		4, 8, centerX, centerY, centerZ, sizeX, sizeY, sizeZ)), 0644)
	if err != nil {
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	cbody, err := cli.ContainerCreate(ctx, &container.Config{
		Cmd: []string{
			"./main.o",
			"--program",
			program,
		},
		Image: image,
		Tty:   true,
	}, &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", dir, "/virtualflow/data"),
		},
	}, nil, nil, "")
	if err != nil {
		return nil, err
	}

	defer func() {
		if err = cli.ContainerStop(ctx, cbody.ID, nil); err != nil {
			return
		}
		if err = cli.ContainerRemove(ctx, cbody.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			return
		}
	}()

	if err := cli.ContainerStart(ctx, cbody.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	out, err := cli.ContainerLogs(ctx, cbody.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
		Details:    true,
	})
	if err != nil {
		return nil, err
	}
	defer out.Close()

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := scanner.Text()
		if progress, err := strconv.ParseFloat(strings.TrimSpace(line), 64); err != nil {
			fmt.Println(line)
		} else {
			onProgress(progress)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if data, err := os.ReadFile(filepath.Join(dir, "err.log")); err != nil {
		return nil, err
	} else if len(data) > 0 {
		return nil, errors.New(string(data))
	}

	outputFile := filepath.Join(dir, "output.pdbqt")
	data, err := os.ReadFile(outputFile)

	if err != nil {
		return nil, err
	}

	return data, err
}
