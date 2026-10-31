import { useEffect, useMemo, useState } from 'react'
import { Button, Drawer, Descriptions, Input, Select, Table, Tag, Space, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { api, Device } from '../api'
import RemoteView from '../components/RemoteView'

const primaryIP = (d: Device) => {
  for (const i of d.interfaces) {
    for (const ip of i.ips) {
      if (!ip.includes(':')) return ip
    }
  }
  return d.interfaces.find((i) => i.ips.length > 0)?.ips[0] ?? '-'
}
const primaryMAC = (d: Device) => d.interfaces.find((i) => i.mac)?.mac ?? '-'
const fmt = (t: string | null) => (t ? new Date(t).toLocaleString('zh-CN') : '-')

export default function Dashboard() {
  const [devices, setDevices] = useState<Device[]>([])
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<string>()
  const [detail, setDetail] = useState<Device | null>(null)
  const [remoteDevice, setRemoteDevice] = useState<Device | null>(null)

  const load = async () => {
    setLoading(true)
    try {
      setDevices(await api<Device[]>('/api/devices'))
    } catch (e) {
      message.error((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    const t = setInterval(load, 15000)
    return () => clearInterval(t)
  }, [])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    return devices.filter((d) => {
      const matchSearch =
        !q ||
        d.hostname.toLowerCase().includes(q) ||
        d.device_id.includes(q) ||
        d.interfaces.some((i) => i.ip.toLowerCase().includes(q) || i.mac.toLowerCase().includes(q))
      const matchStatus = !status || d.status === status
      return matchSearch && matchStatus
    })
  }, [devices, search, status])

  const columns: ColumnsType<Device> = [
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (s: string) =>
        s === 'online' ? <Tag color="green">在线</Tag> : <Tag>离线</Tag>,
    },
    {
      title: '计算机名',
      dataIndex: 'hostname',
      sorter: (a, b) => a.hostname.localeCompare(b.hostname),
    },
    { title: 'IP 地址', render: (_, d) => primaryIP(d) },
    { title: 'MAC 地址', render: (_, d) => primaryMAC(d) },
    {
      title: '系统版本',
      render: (_, d) => `${d.os_name} ${d.os_version}`.trim(),
    },
    { title: 'Build', dataIndex: 'os_build', width: 90 },
    { title: '最近心跳', dataIndex: 'last_seen_at', render: (t) => fmt(t as string | null) },
    {
      title: '操作',
      width: 160,
      render: (_, d) => (
        <Space>
          <Button size="small" onClick={() => setDetail(d)}>
            详情
          </Button>
          <Button
            size="small"
            type="primary"
            disabled={d.status !== 'online'}
            onClick={() => setRemoteDevice(d)}
          >
            远程
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }} wrap>
        <Tag color="blue">{devices.filter((d) => d.status === 'online').length} 在线</Tag>
        <Input.Search
            placeholder="搜索主机名 / IP / MAC"
            allowClear
            style={{ width: 280 }}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Select
            placeholder="状态"
            allowClear
            style={{ width: 120 }}
            onChange={setStatus}
            options={[
              { value: 'online', label: '在线' },
              { value: 'offline', label: '离线' },
            ]}
          />
          <Button onClick={load}>刷新</Button>
        </Space>
        <Table
          rowKey="device_id"
          loading={loading}
          dataSource={filtered}
          columns={columns}
          pagination={{ pageSize: 50, showSizeChanger: false }}
          size="middle"
        />

      <Drawer title={detail?.hostname} open={!!detail} onClose={() => setDetail(null)} width={480}>
        {detail && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="状态">
              {detail.status === 'online' ? <Tag color="green">在线</Tag> : <Tag>离线</Tag>}
            </Descriptions.Item>
            <Descriptions.Item label="主机名">{detail.hostname}</Descriptions.Item>
            <Descriptions.Item label="系统">{detail.os_name}</Descriptions.Item>
            <Descriptions.Item label="版本">{detail.os_version}</Descriptions.Item>
            <Descriptions.Item label="Build">{detail.os_build}</Descriptions.Item>
            <Descriptions.Item label="Agent 版本">{detail.agent_version}</Descriptions.Item>
            <Descriptions.Item label="首次上线">{fmt(detail.first_seen_at)}</Descriptions.Item>
            <Descriptions.Item label="最近心跳">{fmt(detail.last_seen_at)}</Descriptions.Item>
            {detail.interfaces.map((i, idx) => (
              <Descriptions.Item key={idx} label={i.name || `网卡 ${idx + 1}`}>
                {i.mac} — {i.ips.join(', ') || '无 IP'}
              </Descriptions.Item>
            ))}
          </Descriptions>
        )}
      </Drawer>

      <RemoteView device={remoteDevice} onClose={() => setRemoteDevice(null)} />
    </div>
  )
}
