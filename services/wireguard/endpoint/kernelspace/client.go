/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package kernelspace

import (
	"encoding/base64"
	"net"
	"os/exec"

	"github.com/mdlayher/wireguardctrl"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
)

var allowedIPs = []net.IPNet{
	{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
	{IP: net.IPv6zero, Mask: net.CIDRMask(0, 128)},
}

type client struct {
	iface    string
	wgClient *wireguardctrl.Client
}

// NewWireguardClient creates new wireguard kernel space client.
func NewWireguardClient() (*client, error) {
	wgClient, err := wireguardctrl.New()
	if err != nil {
		return nil, err
	}
	return &client{wgClient: wgClient}, nil
}

func (c *client) ConfigureDevice(iface string, config wg.DeviceConfig, subnet net.IPNet) error {
	var deviceConfig wgtypes.Config
	if config != nil {
		port := config.ListenPort()
		privateKey, err := stringToKey(config.PrivateKey())
		if err != nil {
			return err
		}
		deviceConfig.PrivateKey = &privateKey
		deviceConfig.ListenPort = &port
	}

	if err := c.up(iface, subnet); err != nil {
		return err
	}
	c.iface = iface
	return c.wgClient.ConfigureDevice(iface, deviceConfig)
}

func (c *client) AddPeer(iface string, peer wg.PeerInfo) error {
	var deviceConfig wgtypes.Config
	if peer != nil {
		endpoint := peer.Endpoint()
		publicKey, err := stringToKey(peer.PublicKey())
		if err != nil {
			return err
		}
		deviceConfig.Peers = []wgtypes.PeerConfig{{
			Endpoint:   endpoint,
			PublicKey:  publicKey,
			AllowedIPs: allowedIPs,
		}}
	}
	return c.wgClient.ConfigureDevice(iface, deviceConfig)
}

func (c *client) up(iface string, subnet net.IPNet) error {
	if d, err := c.wgClient.Device(iface); err != nil || d.Name != iface {
		if err := exec.Command("ip", "link", "add", "dev", iface, "type", "wireguard").Run(); err != nil {
			return err
		}
	}

	if err := exec.Command("ip", "address", "replace", "dev", iface, subnet.String()).Run(); err != nil {
		return err
	}

	return exec.Command("ip", "link", "set", "dev", iface, "up").Run()
}

func (c *client) Close() error {
	if err := exec.Command("ip", "link", "del", "dev", c.iface).Run(); err != nil {
		return err
	}
	return c.wgClient.Close()
}

// GeneratePrivateKey creates new wireguard private key
func GeneratePrivateKey() (string, error) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", err
	}
	return key.String(), nil
}

// PrivateKeyToPublicKey generates wireguard public key from private key
func PrivateKeyToPublicKey(key string) (string, error) {
	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	privateKey, err := wgtypes.NewKey(k)
	if err != nil {
		return "", err
	}
	return privateKey.PublicKey().String(), nil
}

func stringToKey(key string) (wgtypes.Key, error) {
	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return wgtypes.Key{}, err
	}
	return wgtypes.NewKey(k)
}