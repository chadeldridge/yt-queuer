package application

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	regCECAlias        = regexp.MustCompile(`^[a-zA-Z0-9\. _-]{1,14}$`)
	regCECDevice       = regexp.MustCompile(`^[0-9]$|^cec[0-9]$`)
	regCECPhysicalAddr = regexp.MustCompile(`^[0-9A-F].[0-9A-F].[0-9A-F].[0-9A-F]$`)
)

type CEC struct {
	PBCID        string `json:"id"`
	Alias        string `json:"alias"`         // Max 14 characters
	Device       string `json:"device"`        // -d0 or -device=cec0
	LogicalAddr  int    `json:"logical_addr"`  // -t0 --to=0
	PhysicalAddr string `json:"physical_addr"` // 1.0.0.0
}

func NewCEC(pbcid, alias, device, physicalAddr string, logicalAddr int) (CEC, error) {
	cec := CEC{
		PBCID:        pbcid,
		Alias:        alias,
		Device:       device,
		PhysicalAddr: physicalAddr,
		LogicalAddr:  logicalAddr,
	}

	if err := cec.Validate(); err != nil {
		return CEC{}, err
	}

	return cec, nil
}

func (cec CEC) Validate() error {
	if !regPBCID.MatchString(cec.PBCID) {
		return fmt.Errorf("invalid playback client id")
	}

	if !regCECAlias.MatchString(cec.Alias) {
		return fmt.Errorf("invalid alias: allowed characters (min 1, max 14) a-z, A-Z, 0-9, ., _, -")
	}

	if !regCECDevice.MatchString(cec.Device) {
		return fmt.Errorf("invalid device: must be 0 - 9 or cec0 - cec9")
	}

	if cec.LogicalAddr < 0 || cec.LogicalAddr > 15 {
		return fmt.Errorf("invalid logical address: must be 0 - 15")
	}

	if !regCECPhysicalAddr.MatchString(cec.PhysicalAddr) {
		return fmt.Errorf("invalid physical address: must be 0.0.0.0 - F.F.F.F")
	}

	return nil
}

// TODO: Look into using a go library for CEC instead of shelling out.

// TODO: Add monitoring of CEC events. Need to look for:
// Received from TV to all (0 to 15): REQUEST_ACTIVE_SOURCE (0x85)
// This will let us know when to tell the TV to switch inputs to the devices physical address after
// a power on request.

/*
func (cec CEC) Monitor() error {
	args := []string{"-d", cec.Device, "-t", strconv.Itoa(cec.LogicalAddr), "--monitor"}
	cmd := exec.Command("/usr/bin/sudo /usr/bin/cec-ctl", args...)
	return err
}
*/

func (cec CEC) run(args []string) (string, error) {
	a := []string{"-d", cec.Device, "-t", strconv.Itoa(cec.LogicalAddr)}
	a = append(a, args...)
	cmd := exec.Command("/usr/bin/cec-ctl", a...)

	// Capture output
	var out bytes.Buffer
	var stdErr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stdErr

	err := cmd.Run()
	if err != nil {
		err := fmt.Errorf("/usr/bin/cec-ctl %s: %w: %s", strings.Join(a, " "), err, stdErr.String())
		return "", err
	}

	return out.String(), nil
}

var (
	regStatus      = regexp.MustCompile(`pwr-state: ([a-z]*)`)
	ErrCECNoStatus = fmt.Errorf("no power status found")
)

// PowerStatus sends a CEC command to get the power status of the remote device.
func (cec CEC) PowerStatus() (string, error) {
	out, err := cec.run([]string{"--give-device-power-status"})
	if err != nil {
		return "", err
	}

	m := regStatus.FindStringSubmatch(out)
	if len(m) < 2 {
		return "", ErrCECNoStatus
	}

	return m[1], nil
}

// PowerOn sends a CEC command to power on the remote device and switch it's input to this device.
func (cec CEC) PowerOn() error {
	_, err := cec.run([]string{"--image-view-on"})
	if err != nil {
		return fmt.Errorf("failed to power on: %w", err)
	}

	// We need to add monitoring of CEC events to know when to switch the input to the device.
	/*
		time.Sleep(1 * time.Second)
		for i := 0; i < 5; i++ {
			status, err := cec.PowerStatus()
			if err != nil && err != ErrCECNoStatus {
				return fmt.Errorf("failed to get power status: %w", err)
			}

			log.Printf("Power status: %s", status)
			if status == "on" {
				break
			}

			time.Sleep(1 * time.Second)
		}

		time.Sleep(1 * time.Second)
		_, err = cec.run([]string{"--active-source", "phys-addr=" + cec.PhysicalAddr})
	*/
	return err
}

// PowerOff sends a CEC command to power off the remote device.
func (cec CEC) PowerOff() error {
	_, err := cec.run([]string{"--standby"})
	return err
}
