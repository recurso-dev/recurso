import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import posthog from 'posthog-js'
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

    // Identify the signed-in tenant/user to PostHog (inert without
    // VITE_POSTHOG_KEY) so product-analytics events tie to a tenant. Runs
    // whenever auth resolves or changes.
    useEffect(() => {
        if (import.meta.env.VITE_POSTHOG_KEY && user) {
            posthog.identify(user.id, { email: user.email, tenant_id: user.tenant_id })
        }
    }, [user])

    // On load, resolve the session cookie via /auth/me. If there's no session
    // but a stored API key exists, we stay authenticated in legacy mode.
    // ?demo=1 (the public sandbox entry) first opens a demo session via
    // POST /auth/demo — the endpoint only exists when the server runs
    // DEMO_MODE, so this is inert everywhere else.
    useEffect(() => {
        let active = true
        const params = new URLSearchParams(window.location.search)
        const wantsDemo = params.get('demo') === '1'

        // Only a 401 means "no session". Transient failures (rate limit, cold
        // start, network blip) get two retries with backoff — without this,
        // any hiccup bounced a validly-logged-in user to the login screen.
        const resolve = async () => {
            for (let attempt = 0; attempt < 3; attempt++) {
                try {
                    const res = await endpoints.authMe()
                    if (active) setUser(res.data?.user || null)
                    break
                } catch (err) {
                    const status = err?.response?.status
                    if (status === 401 || attempt === 2) {
                        if (active) setUser(null)
                        break
                    }
                    await new Promise((r) => setTimeout(r, 750 * (attempt + 1)))
                    if (!active) break
                }
            }
            if (active) setLoading(false)
        }

        if (wantsDemo) {
            endpoints
                .authDemo()
                .catch(() => {})
                .finally(resolve)
        } else {
            resolve()
        }
        return () => {
            active = false
        }
    }, [])

    // All auth actions are useCallback-stable and the context value is memoized
    // (see the `value` useMemo below). This keeps the context reference constant
    // across unrelated re-renders, so an effect that depends on one of these
    // functions can't be re-run mid-request — the root cause of the verify-email
    // spinner hang (#342). Behaviour is otherwise unchanged.

    // Email/password login → httpOnly session cookie.
    const login = useCallback(async (email, password) => {
        const res = await endpoints.authLogin(email, password)
        setUser(res.data?.user || null)
        return res.data
    }, [])

    // Second step of two-step login: exchange the mfa_token + code for a
    // session cookie. Bad codes throw (401) for the caller to surface.
    const loginMfa = useCallback(async (mfaToken, code) => {
        const res = await endpoints.loginMfa(mfaToken, code)
        setUser(res.data?.user || null)
        return res.data
    }, [])

    // Register a new tenant + owner user; the server opens a session.
    const registerAccount = useCallback(async (data) => {
        const res = await endpoints.authRegister(data)
        setUser(res.data?.user || null)
        return res.data
    }, [])

    // Legacy: authenticate by pasting a tenant API key (Bearer). Held in memory
    // only — not persisted to localStorage (XSS hardening).
    const loginWithApiKey = useCallback((key) => {
        storeApiKey(key)
        setApiKeyState(key)
    }, [])

    // userRef mirrors the latest user so refreshUser can stay useCallback-stable
    // while still returning the current user on a best-effort failure.
    const userRef = useRef(user)
    userRef.current = user

    // Re-resolve the current user from /auth/me (e.g. after email verification
    // so the verify banner clears). Best-effort: a failure leaves state as-is.
    const refreshUser = useCallback(async () => {
        try {
            const res = await endpoints.authMe()
            setUser(res.data?.user || null)
            return res.data?.user || null
        } catch {
            return userRef.current
        }
    }, [])

    const logout = useCallback(async () => {
        try {
            await endpoints.authLogout()
        } catch {
            // ignore — clear locally regardless
        }
        clearApiKey()
        setApiKeyState('')
        setUser(null)
    }, [])

    const value = useMemo(
        () => ({
            user,
            apiKey,
            isAuthenticated: !!user || !!apiKey,
            loading,
            login,
            loginMfa,
            registerAccount,
            loginWithApiKey,
            logout,
            refreshUser,
        }),
        [user, apiKey, loading, login, loginMfa, registerAccount, loginWithApiKey, logout, refreshUser]
    )

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export const useAuth = () => useContext(AuthContext)
