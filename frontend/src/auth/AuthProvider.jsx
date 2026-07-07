import React, { createContext, useContext, useState, useEffect } from 'react'
import { endpoints } from '../lib/api'

const AuthContext = createContext(null)

export const AuthProvider = ({ children }) => {
    const [user, setUser] = useState(null)
    // Legacy API-key mode (dev / programmatic) coexists with cookie sessions.
    const [apiKey, setApiKey] = useState(() => localStorage.getItem('recurso_api_key') || '')
    const [loading, setLoading] = useState(true)

    // On load, resolve the session cookie via /auth/me. If there's no session
    // but a stored API key exists, we stay authenticated in legacy mode.
    useEffect(() => {
        let active = true
        endpoints
            .authMe()
            .then((res) => {
                if (active) setUser(res.data?.user || null)
            })
            .catch(() => {
                if (active) setUser(null)
            })
            .finally(() => {
                if (active) setLoading(false)
            })
        return () => {
            active = false
        }
    }, [])

    // Email/password login → httpOnly session cookie.
    const login = async (email, password) => {
        const res = await endpoints.authLogin(email, password)
        setUser(res.data?.user || null)
        return res.data
    }

    // Register a new tenant + owner user; the server opens a session.
    const registerAccount = async (data) => {
        const res = await endpoints.authRegister(data)
        setUser(res.data?.user || null)
        return res.data
    }

    // Legacy: authenticate by pasting a tenant API key (Bearer).
    const loginWithApiKey = (key) => {
        localStorage.setItem('recurso_api_key', key)
        setApiKey(key)
    }

    const logout = async () => {
        try {
            await endpoints.authLogout()
        } catch {
            // ignore — clear locally regardless
        }
        localStorage.removeItem('recurso_api_key')
        setApiKey('')
        setUser(null)
    }

    const isAuthenticated = !!user || !!apiKey

    return (
        <AuthContext.Provider
            value={{ user, apiKey, isAuthenticated, loading, login, registerAccount, loginWithApiKey, logout }}
        >
            {children}
        </AuthContext.Provider>
    )
}

export const useAuth = () => useContext(AuthContext)
