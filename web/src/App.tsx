import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import type { ReactElement } from 'react'
import Login from './pages/Login'
import MainLayout from './layouts/MainLayout'
import Home from './pages/Home'
import Dashboard from './pages/Dashboard'
import Command from './pages/Command'
import Action from './pages/Action'
import { getToken } from './api'

function RequireAuth({ children }: { children: ReactElement }) {
  return getToken() ? children : <Navigate to="/login" replace />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route
          path="/"
          element={
            <RequireAuth>
              <MainLayout />
            </RequireAuth>
          }
        >
          <Route index element={<Navigate to="/home" replace />} />
          <Route path="home" element={<Home />} />
          <Route path="intel/device-control" element={<Dashboard />} />
          <Route path="command" element={<Command />} />
          <Route path="action" element={<Action />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
