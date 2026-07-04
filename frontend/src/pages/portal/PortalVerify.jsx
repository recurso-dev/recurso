import React, { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { API_ROOT as API_BASE } from '../../lib/api'

const PortalVerify = () => {
    const [searchParams] = useSearchParams()
    const navigate = useNavigate()
    const token = searchParams.get('token')

    useEffect(() => {
        if (!token) {
            navigate('/portal/login')
            return
        }

        const verifyToken = async () => {
            try {
                const response = await fetch(`${API_BASE}/portal/auth/verify?token=${token}`)
                const data = await response.json()

                if (response.ok && data.session_token) {
                    localStorage.setItem('portal_session', data.session_token)
                    navigate('/portal/dashboard')
                } else {
                    navigate('/portal/login?error=invalid')
                }
            } catch (err) {
                navigate('/portal/login?error=network')
            }
        }

        verifyToken()
    }, [token, navigate])

    return (
        <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-950">
            <div className="text-center">
                <div className="w-12 h-12 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4" />
                <p className="text-slate-600 dark:text-slate-400">Verifying your login...</p>
            </div>
        </div>
    )
}

export default PortalVerify
