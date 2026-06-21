import React, { createContext, useContext, useState, useEffect } from 'react'

const AuthContext = createContext(null)

export const AuthProvider = ({ children }) => {
    const [apiKey, setApiKey] = useState(() => localStorage.getItem('recurso_api_key') || '')

    const login = (key) => {
        localStorage.setItem('recurso_api_key', key)
        setApiKey(key)
    }

    const logout = () => {
        localStorage.removeItem('recurso_api_key')
        setApiKey('')
    }

    return (
        <AuthContext.Provider value={{ apiKey, isAuthenticated: !!apiKey, login, logout }}>
            {children}
        </AuthContext.Provider>
    )
}

export const useAuth = () => useContext(AuthContext)
