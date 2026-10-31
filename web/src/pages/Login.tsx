import { useState } from 'react'
import { Button, Card, Form, Input, message } from 'antd'
import { useNavigate } from 'react-router-dom'
import { api, setToken } from '../api'

export default function Login() {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const onFinish = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      const data = await api<{ token: string }>('/api/login', {
        method: 'POST',
        body: JSON.stringify(values),
      })
      setToken(data.token)
      message.success('登录成功')
      navigate('/')
    } catch (e) {
      message.error((e as Error).message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
        background: '#f0f2f5',
      }}
    >
      <Card title="情指行一体化平台" style={{ width: 360 }}>
        <Form onFinish={onFinish} layout="vertical">
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input placeholder="admin" autoFocus />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password />
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={loading}>
            登录
          </Button>
        </Form>
      </Card>
    </div>
  )
}
