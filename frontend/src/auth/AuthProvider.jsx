import React, { createContext, useContext, useState, useEffect } from 'react'
import { endpoints } from '../lib/api'
import { getApiKey, setApiKey as storeApiKey, clearApiKey } from '../lib/authToken'

const AuthContext = createContext(null)

export const AuthProvider = ({ children }) => {
    const [user, setUser] = useState(null)
    // Legacy API-key mode (dev / programmatic) coexists with cookie sessions.
    // The key is held in memory (lib/authToken.js), never in localStorage, so an
    // XSS payload can't lift it from storage; it clears on refresh.
    const [apiKey, setApiKeyState] = useState(() => getApiKey())
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

    // Second step of two-step login: exchange the mfa_token + code for a
    // session cookie. Bad codes throw (401) for the caller to surface.
    const loginMfa = async (mfaToken, code) => {
        const res = await endpoints.loginMfa(mfaToken, code)
        setUser(res.data?.user || null)
        return res.data
    }

    // Register a new tenant + owner user; the server opens a session.
    const registerAccount = async (data) => {
        const res = await endpoints.authRegister(data)
        setUser(res.data?.user || null)
        return res.data
    }

    // Legacy: authenticate by pasting a tenant API key (Bearer). Held in memory
    // only — not persisted to localStorage (XSS hardening).
    const loginWithApiKey = (key) => {
        storeApiKey(key)
        setApiKeyState(key)
    }

    const logout = async () => {
        try {
            await endpoints.authLogout()
        } catch {
            // ignore — clear locally regardless
        }
        clearApiKey()
        setApiKeyState('')
        setUser(null)
    }

    const isAuthenticated = !!user || !!apiKey

    return (
        <AuthContext.Provider
            value={{ user, apiKey, isAuthenticated, loading, login, loginMfa, registerAccount, loginWithApiKey, logout }}
        >
            {children}
        </AuthContext.Provider>
    )
}

export const useAuth = () => useContext(AuthContext)
