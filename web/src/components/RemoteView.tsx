import { useEffect, useRef, useState } from 'react'
import { Button, Modal, Select, Space, Switch, message } from 'antd'
// @ts-ignore - @novnc/novnc ships no bundled type declarations
import RFB from '@novnc/novnc'
import { api, Device } from '../api'

export default function RemoteView({
  device,
  onClose,
}: {
  device: Device | null
  onClose: () => void
}) {
  const screenRef = useRef<HTMLDivElement>(null)
  const rfbRef = useRef<any>(null)
  const [viewOnly, setViewOnly] = useState(false)
  const [quality, setQuality] = useState<'high' | 'medium' | 'low'>('high')

  useEffect(() => {
    if (!device) return
    let disposed = false

    ;(async () => {
      try {
        const scale = { high: 1, medium: 0.5, low: 0.25 }[quality]
        const data = await api<{ session_id: string }>(
          `/api/devices/${device.device_id}/remote`,
          { method: 'POST', body: JSON.stringify({ scale }) },
        )
        if (disposed) return
        const proto = location.protocol === 'https:' ? 'wss' : 'ws'
        const url = `${proto}://${location.host}/ws/remote?session=${data.session_id}`
        const rfb = new RFB(screenRef.current, url)
        rfbRef.current = rfb
        rfb.viewOnly = viewOnly
        rfb.scaleViewport = true
        rfb.resizeSession = false
        rfb.showDotCursor = false
        rfb.addEventListener('disconnect', (e: any) => {
          if (!e.detail.clean) {
            message.error('远程连接已断开')
          }
          onClose()
        })
      } catch (e) {
        if (!disposed) {
          message.error((e as Error).message)
          onClose()
        }
      }
    })()

    return () => {
      disposed = true
      if (rfbRef.current) {
        try {
          rfbRef.current.disconnect()
        } catch {
          /* ignore */
        }
        rfbRef.current = null
      }
      api(`/api/devices/${device.device_id}/remote`, { method: 'DELETE' }).catch(() => {})
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [device, quality])

  useEffect(() => {
    if (rfbRef.current) rfbRef.current.viewOnly = viewOnly
  }, [viewOnly])

  return (
    <Modal
      title={`远程控制 — ${device?.hostname ?? ''}`}
      open={!!device}
      onCancel={onClose}
      footer={null}
      width="92%"
      style={{ top: 20 }}
    >
      <Space style={{ marginBottom: 12 }}>
        <span>画质</span>
        <Select
          value={quality}
          onChange={setQuality}
          style={{ width: 110 }}
          options={[
            { value: 'high', label: '高清' },
            { value: 'medium', label: '标清' },
            { value: 'low', label: '流畅' },
          ]}
        />
        <Switch checked={viewOnly} onChange={setViewOnly} />
        <span>仅查看（关闭后键鼠可接管）</span>
        <Button danger onClick={onClose}>
          断开连接
        </Button>
      </Space>
      <div
        style={{ background: '#000', display: 'flex', justifyContent: 'center', minHeight: 480 }}
      >
        <div
          ref={screenRef}
          style={{ width: '86vw', height: '72vh', maxWidth: 1600 }}
        />
      </div>
    </Modal>
  )
}
