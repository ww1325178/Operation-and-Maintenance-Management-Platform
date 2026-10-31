import { useState } from 'react'
import { Layout, Menu, Button, Space } from 'antd'
import {
  HomeOutlined,
  RadarChartOutlined,
  ControlOutlined,
  ThunderboltOutlined,
  DesktopOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { clearToken } from '../api'

const menuItems = [
  { key: '/home', icon: <HomeOutlined />, label: '首页' },
  {
    key: 'intel',
    icon: <RadarChartOutlined />,
    label: '情报',
    children: [
      {
        key: '/intel/device-control',
        icon: <DesktopOutlined />,
        label: '一机两用及上网行为管控',
      },
    ],
  },
  { key: '/command', icon: <ControlOutlined />, label: '指挥' },
  { key: '/action', icon: <ThunderboltOutlined />, label: '行动' },
]

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()

  const selectedKey = location.pathname.startsWith('/intel/')
    ? '/intel/device-control'
    : location.pathname

  const [openKeys, setOpenKeys] = useState<string[]>(() =>
    location.pathname.startsWith('/intel/') ? ['intel'] : [],
  )
  const [collapsed, setCollapsed] = useState(false)

  const pageTitle = location.pathname === '/home'
    ? '首页'
    : location.pathname.startsWith('/intel/')
      ? '情报 · 一机两用及上网行为管控'
      : location.pathname === '/command'
        ? '指挥'
        : location.pathname === '/action'
          ? '行动'
          : '情指行一体化平台'

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Sider
        width={248}
        collapsible
        collapsed={collapsed}
        onCollapse={setCollapsed}
        trigger={null}
        theme="dark"
      >
        <div
          style={{
            height: 56,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#fff',
            fontSize: 16,
            fontWeight: 600,
            letterSpacing: 1,
          }}
        >
          {collapsed ? '情指' : '情指行一体化平台'}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          openKeys={openKeys}
          onOpenChange={(keys) => setOpenKeys(keys as string[])}
          items={menuItems}
          onClick={({ key }) => {
            if (key.startsWith('/')) navigate(key)
          }}
        />
      </Layout.Sider>
      <Layout>
        <Layout.Header
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            background: '#fff',
            borderBottom: '1px solid #f0f0f0',
            padding: '0 24px',
          }}
        >
          <Space size="middle">
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed(!collapsed)}
            />
            <span style={{ fontSize: 16, fontWeight: 600 }}>{pageTitle}</span>
          </Space>
          <Space>
            <Button
              onClick={() => {
                clearToken()
                window.location.href = '/login'
              }}
            >
              退出登录
            </Button>
          </Space>
        </Layout.Header>
        <Layout.Content style={{ padding: 24, background: '#f5f5f5' }}>
          <Outlet />
        </Layout.Content>
      </Layout>
    </Layout>
  )
}
