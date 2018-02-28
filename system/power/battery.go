/*
 * Copyright (C) 2016 ~ 2018 Deepin Technology Co., Ltd.
 *
 * Author:     jouyouyun <jouyouwen717@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package power

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gir/gudev-1.0"
	"pkg.deepin.io/dde/api/powersupply/battery"
	"pkg.deepin.io/lib/dbusutil"
)

type Battery struct {
	service *dbusutil.Service
	exit    chan struct{}
	mutex   sync.Mutex

	gudevClient       *gudev.Client
	changedProperties []string

	PropsMu   sync.RWMutex
	SysfsPath string
	IsPresent bool

	Manufacturer string
	ModelName    string
	SerialNumber string
	Name         string
	Technology   string

	Energy           float64
	EnergyFull       float64
	EnergyFullDesign float64
	EnergyRate       float64

	Voltage     float64
	Percentage  float64
	Capacity    float64
	Status      battery.Status
	TimeToEmpty uint64
	TimeToFull  uint64
	UpdateTime  int64

	refreshDone func()

	methods *struct {
		Debug func() `in:"cmd"`
	}
}

func newBattery(manager *Manager, device *gudev.Device) *Battery {
	sysfsPath := device.GetSysfsPath()
	logger.Debugf("NewBattery sysfsPath: %q", sysfsPath)
	if manager == nil || manager.gudevClient == nil ||
		device == nil {
		return nil
	}
	bat := &Battery{
		service:     manager.service,
		gudevClient: manager.gudevClient,
		SysfsPath:   sysfsPath,
	}
	bat.refresh(device)
	bat.resetUpdateInterval(60 * time.Second)
	return bat
}

const (
	batteryDBusIFC = dbusIFC + ".Battery"
)

func (bat *Battery) GetDBusExportInfo() dbusutil.ExportInfo {
	return dbusutil.ExportInfo{
		Path:      dbusPath + "/battery_" + getValidName(filepath.Base(bat.SysfsPath)),
		Interface: batteryDBusIFC,
	}
}

func getValidName(n string) string {
	// dbus objpath 0-9 a-z A-Z _
	n = strings.Replace(n, "-", "_x0", -1)
	n = strings.Replace(n, ".", "_x1", -1)
	n = strings.Replace(n, ":", "_x2", -1)
	return n
}

func (bat *Battery) setRefreshDoneCallback(fn func()) {
	bat.refreshDone = fn
}

func (bat *Battery) newDevice() *gudev.Device {
	return bat.gudevClient.QueryBySysfsPath(bat.SysfsPath)
}

func (bat *Battery) notifyChange(propNames ...string) {
	bat.changedProperties = append(bat.changedProperties, propNames...)
}

func (bat *Battery) refresh(dev *gudev.Device) {
	endDelay := bat.service.DelayEmitPropertyChanged(bat)
	batInfo := battery.GetBatteryInfo(dev)
	bat._refresh(batInfo)
	if endDelay != nil {
		endDelay()
	}
}

func (bat *Battery) _refresh(info *battery.BatteryInfo) {
	logger.Debug("Refresh", bat.Name)
	isPresent := true
	var updateTime int64
	if info == nil {
		isPresent = false
		info = &battery.BatteryInfo{}
	} else {
		now := time.Now()
		updateTime = now.Unix()
		logger.Debugf("now %v updateTime %v", now, updateTime)
	}

	logger.Debug("Name", info.Name)
	logger.Debug("Technology", info.Technology)
	logger.Debug("Manufacturer", info.Manufacturer)
	logger.Debug("ModelName", info.ModelName)
	logger.Debug("SerialNumber", info.SerialNumber)
	logger.Debugf("energy %v", info.Energy)
	logger.Debugf("energyFull %v", info.EnergyFull)
	logger.Debugf("EnergyFullDesign %v", info.EnergyFullDesign)
	logger.Debugf("EnergyRate %v", info.EnergyRate)
	logger.Debugf("voltage %v", info.Voltage)
	logger.Debugf("percentage %.4f%%", info.Percentage)
	logger.Debugf("capacity %.4f%%", info.Capacity)
	logger.Debug("status", info.Status)
	logger.Debugf("timeToEmpty %v (%vs), timeToFull %v (%vs)",
		time.Duration(info.TimeToEmpty)*time.Second,
		info.TimeToEmpty,
		time.Duration(info.TimeToFull)*time.Second,
		info.TimeToFull)

	bat.PropsMu.Lock()
	bat.setPropIsPresent(isPresent)
	bat.setPropUpdateTime(updateTime)
	bat.setPropName(info.Name)
	bat.setPropTechnology(info.Technology)
	bat.setPropManufacturer(info.Manufacturer)
	bat.setPropModelName(info.ModelName)
	bat.setPropSerialNumber(info.SerialNumber)
	bat.setPropEnergy(info.Energy)
	bat.setPropEnergyFull(info.EnergyFull)
	bat.setPropEnergyFullDesign(info.EnergyFullDesign)
	bat.setPropEnergyRate(info.EnergyRate)
	bat.setPropVoltage(info.Voltage)
	bat.setPropPercentage(info.Percentage)
	bat.setPropCapacity(info.Capacity)
	bat.setPropStatus(info.Status)
	bat.setPropTimeToEmpty(info.TimeToEmpty)
	bat.setPropTimeToFull(info.TimeToFull)
	bat.PropsMu.Unlock()

	logger.Debugf("Refresh %v done", bat.Name)
	if bat.refreshDone != nil {
		bat.refreshDone()
	}
}

func (bat *Battery) Refresh() {
	dev := bat.newDevice()
	if dev != nil {
		bat.refresh(dev)
		dev.Unref()
	} else {
		logger.Warningf("Refresh %v failed", bat.Name)
	}
}

func (bat *Battery) startLoopUpdate(d time.Duration) chan struct{} {
	done := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(d)
		defer ticker.Stop()
		for {
			select {
			case _, ok := <-ticker.C:
				if !ok {
					logger.Error("Invalid ticker event")
					return
				}

				bat.Refresh()
			case <-done:
				return
			}
		}
	}()
	return done
}

func (bat *Battery) resetUpdateInterval(d time.Duration) {
	if bat.exit != nil {
		close(bat.exit)
	}
	bat.exit = bat.startLoopUpdate(d)
}

func (bat *Battery) destroy() {
	if bat.exit != nil {
		close(bat.exit)
		bat.exit = nil
	}
}
