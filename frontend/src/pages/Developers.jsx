import React, { useState, useEffect } from 'react'
import { endpoints } from '../lib/api'
import Modal from '../components/Modal'

const Developers = () => {
    const [keys, setKeys] = useState([])
    const [webhooks, setWebhooks] = useState([])
    const [events, setEvents] = useState([])
    const [eventTypes, setEventTypes] = useState([])
    const [activeTab, setActiveTab] = useState('keys') // 'keys', 'webhooks', 'events'
    const [loading, setLoading] = useState(true)
    const [generatedKey, setGeneratedKey] = useState(null)
    const [isModalOpen, setIsModalOpen] = useState(false)
    const [isWebhookModalOpen, setIsWebhookModalOpen] = useState(false)
    const [newWebhook, setNewWebhook] = useState({ url: '', events: [] })
    const [createdWebhookSecret, setCreatedWebhookSecret] = useState(null)

    const fetchKeys = async () => {
        try {
            const response = await endpoints.getAPIKeys()
            setKeys(response.data.data || [])
        } catch (error) {
            console.error(error)
        } finally {
            setLoading(false)
        }
    }

    const fetchWebhooks = async () => {
        try {
            const response = await endpoints.getWebhooks()
            setWebhooks(response.data.data || [])
        } catch (error) {
            console.error('Failed to fetch webhooks:', error)
        }
    }

    const fetchEvents = async () => {
        try {
            const response = await endpoints.getEvents({ limit: 50 })
            setEvents(response.data.data || [])
        } catch (error) {
            console.error('Failed to fetch events:', error)
        }
    }

    const fetchEventTypes = async () => {
        try {
            const response = await endpoints.getEventTypes()
            setEventTypes(response.data.data || [])
        } catch (error) {
            console.error('Failed to fetch event types:', error)
        }
    }

    useEffect(() => {
        fetchKeys()
        fetchWebhooks()
        fetchEvents()
        fetchEventTypes()
    }, [])

    const handleCreateKey = async () => {
        try {
            const response = await endpoints.createKey({})
            setGeneratedKey(response.data.key)
            setIsModalOpen(true)
            fetchKeys()
        } catch (error) {
            console.error("Failed to create key:", error)
        }
    }

    const handleCreateWebhook = async () => {
        if (!newWebhook.url || newWebhook.events.length === 0) {
            alert('Please enter a URL and select at least one event type.')
            return
        }
        try {
            const response = await endpoints.createWebhook(newWebhook)
            setCreatedWebhookSecret(response.data.data?.secret)
            setNewWebhook({ url: '', events: [] })
            fetchWebhooks()
        } catch (error) {
            console.error("Failed to create webhook:", error)
            alert('Failed to create webhook: ' + (error.response?.data?.error?.message || error.message))
        }
    }

    const handleDeleteWebhook = async (id) => {
        if (!confirm('Are you sure you want to delete this webhook endpoint?')) return
        try {
            await endpoints.deleteWebhook(id)
            fetchWebhooks()
        } catch (error) {
            console.error("Failed to delete webhook:", error)
        }
    }

    const toggleEventType = (eventType) => {
        setNewWebhook(prev => {
            const events = prev.events.includes(eventType)
                ? prev.events.filter(e => e !== eventType)
                : [...prev.events, eventType]
            return { ...prev, events }
        })
    }

    return (
        <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
            <header className="flex flex-wrap items-center justify-between gap-4">
                <div className="flex flex-col gap-1">
                    <h1 className="text-slate-900 dark:text-white text-3xl font-bold tracking-tight">Developer Settings</h1>
                    <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal">Manage your API keys, webhooks, and view event logs.</p>
                </div>
                {activeTab === 'keys' && (
                    <button
                        onClick={handleCreateKey}
                        className="flex items-center justify-center gap-2 overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-semibold leading-normal shadow-sm hover:bg-primary/90 transition-all"
                    >
                        <span className="material-symbols-outlined text-lg">add</span>
                        <span className="truncate">Create API Key</span>
                    </button>
                )}
                {activeTab === 'webhooks' && (
                    <button
                        onClick={() => setIsWebhookModalOpen(true)}
                        className="flex items-center justify-center gap-2 overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-semibold leading-normal shadow-sm hover:bg-primary/90 transition-all"
                    >
                        <span className="material-symbols-outlined text-lg">add</span>
                        <span className="truncate">Add Endpoint</span>
                    </button>
                )}
            </header>

            {/* Tabs */}
            <div className="mt-8">
                <div className="flex border-b border-slate-200 dark:border-slate-800 gap-8">
                    <button
                        onClick={() => setActiveTab('keys')}
                        className={`flex flex-col items-center justify-center border-b-[3px] pb-[13px] pt-2 transition-colors ${activeTab === 'keys' ? 'border-primary text-primary dark:text-white' : 'border-transparent text-slate-500 dark:text-slate-400 hover:border-slate-300 dark:hover:border-slate-700'}`}
                    >
                        <p className="text-sm font-semibold leading-normal">API Keys</p>
                    </button>
                    <button
                        onClick={() => setActiveTab('webhooks')}
                        className={`flex flex-col items-center justify-center border-b-[3px] pb-[13px] pt-2 transition-colors ${activeTab === 'webhooks' ? 'border-primary text-primary dark:text-white' : 'border-transparent text-slate-500 dark:text-slate-400 hover:border-slate-300 dark:hover:border-slate-700'}`}
                    >
                        <p className="text-sm font-semibold leading-normal">Webhooks</p>
                    </button>
                    <button
                        onClick={() => setActiveTab('events')}
                        className={`flex flex-col items-center justify-center border-b-[3px] pb-[13px] pt-2 transition-colors ${activeTab === 'events' ? 'border-primary text-primary dark:text-white' : 'border-transparent text-slate-500 dark:text-slate-400 hover:border-slate-300 dark:hover:border-slate-700'}`}
                    >
                        <p className="text-sm font-semibold leading-normal">Event Logs</p>
                    </button>
                </div>
            </div>

            {/* Content Switcher */}
            <div className="mt-8">
                {/* API Keys Section */}
                {activeTab === 'keys' && (
                    <section>
                        <h2 className="text-slate-800 dark:text-white text-xl font-semibold leading-tight">API Keys</h2>
                        <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal mt-1">Manage and rotate the API keys used to authenticate your API requests.</p>

                        <div className="mt-6 flex flex-col gap-4">
                            {/* Header */}
                            <div className="hidden md:grid grid-cols-12 gap-4 px-4">
                                <div className="col-span-4 text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Key Prefix</div>
                                <div className="col-span-3 text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Type</div>
                                <div className="col-span-2 text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Status</div>
                                <div className="col-span-2 text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">Created</div>
                                <div className="col-span-1"></div>
                            </div>

                            {loading ? (
                                <div className="text-center py-8 text-slate-500">Loading keys...</div>
                            ) : keys.length === 0 ? (
                                <div className="text-center py-8 text-slate-500 bg-slate-50 dark:bg-slate-800/20 rounded-lg border border-dashed border-slate-300 dark:border-slate-700">No API keys found. Generate one to get started.</div>
                            ) : (
                                keys.map((k) => (
                                    <div key={k.id} className="grid grid-cols-1 md:grid-cols-12 items-center gap-4 rounded-lg border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900 shadow-sm transition-all hover:border-primary/50">
                                        <div className="col-span-12 md:col-span-4 flex flex-col">
                                            <span className="md:hidden text-xs text-slate-500 uppercase font-bold mb-1">Key</span>
                                            <code className="font-mono text-sm text-slate-800 dark:text-slate-200 bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded w-fit">{k.key}</code>
                                        </div>
                                        <div className="col-span-6 md:col-span-3 text-sm text-slate-500 dark:text-slate-400">
                                            <span className="md:hidden text-xs text-slate-500 uppercase font-bold mr-2">Type:</span>
                                            Standard Key
                                        </div>
                                        <div className="col-span-6 md:col-span-2">
                                            <span className="md:hidden text-xs text-slate-500 uppercase font-bold mr-2">Status:</span>
                                            <span className="inline-flex items-center rounded-full bg-green-100 px-2 py-1 text-xs font-medium text-green-700 dark:bg-green-900/40 dark:text-green-400">Active</span>
                                        </div>
                                        <div className="col-span-12 md:col-span-2 text-sm text-slate-500 dark:text-slate-400">
                                            <span className="md:hidden text-xs text-slate-500 uppercase font-bold mr-2">Created:</span>
                                            Just now
                                        </div>
                                        <div className="col-span-12 md:col-span-1 flex justify-end">
                                            <button className="text-slate-400 hover:text-red-500 transition-colors">
                                                <span className="material-symbols-outlined">delete</span>
                                            </button>
                                        </div>
                                    </div>
                                ))
                            )}
                        </div>
                    </section>
                )}

                {/* Webhooks Section */}
                {activeTab === 'webhooks' && (
                    <section>
                        <h2 className="text-slate-800 dark:text-white text-xl font-semibold leading-tight">Webhooks</h2>
                        <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal mt-1">Configure endpoints to receive real-time events from Recurso.</p>

                        <div className="mt-6 flex flex-col gap-4">
                            {webhooks.length === 0 ? (
                                <div className="text-center py-8 text-slate-500 bg-slate-50 dark:bg-slate-800/20 rounded-lg border border-dashed border-slate-300 dark:border-slate-700">
                                    No webhook endpoints configured. Add one to receive real-time events.
                                </div>
                            ) : (
                                webhooks.map((hook) => (
                                    <div key={hook.id} className="grid grid-cols-1 md:grid-cols-12 items-start gap-4 rounded-lg border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900 shadow-sm transition-all hover:border-primary/50">
                                        <div className="col-span-12 md:col-span-5 flex flex-col gap-1">
                                            <div className="flex items-center gap-2">
                                                <span className="material-symbols-outlined text-slate-400 text-lg">webhook</span>
                                                <code className="font-mono text-sm font-semibold text-slate-900 dark:text-white break-all">{hook.url}</code>
                                            </div>
                                            <div className="flex flex-wrap gap-2 mt-1">
                                                {hook.events?.map(e => (
                                                    <span key={e} className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-400">{e}</span>
                                                ))}
                                            </div>
                                        </div>
                                        <div className="col-span-6 md:col-span-4 flex flex-col gap-1">
                                            <p className="text-xs text-slate-500 uppercase tracking-wide">Signing Secret</p>
                                            <code className="font-mono text-xs text-slate-400">whsec_•••••••</code>
                                        </div>
                                        <div className="col-span-6 md:col-span-2">
                                            <span className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${hook.status === 'active' ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400' : 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'}`}>
                                                {hook.status?.charAt(0).toUpperCase() + hook.status?.slice(1)}
                                            </span>
                                        </div>
                                        <div className="col-span-12 md:col-span-1 flex justify-end">
                                            <button
                                                onClick={() => handleDeleteWebhook(hook.id)}
                                                className="text-slate-400 hover:text-red-500 transition-colors"
                                            >
                                                <span className="material-symbols-outlined">delete</span>
                                            </button>
                                        </div>
                                    </div>
                                ))
                            )}
                        </div>
                    </section>
                )}

                {/* Event Logs Section */}
                {activeTab === 'events' && (
                    <section>
                        <h2 className="text-slate-800 dark:text-white text-xl font-semibold leading-tight">Event Logs</h2>
                        <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal mt-1">View the history of events generated by your account.</p>

                        <div className="mt-6 flex flex-col rounded-lg border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
                            {events.length === 0 ? (
                                <div className="text-center py-8 text-slate-500">
                                    No events yet. Events will appear here when billing actions occur.
                                </div>
                            ) : (
                                <div className="overflow-x-auto">
                                    <table className="w-full text-left text-sm">
                                        <thead className="border-b border-slate-200 bg-slate-50 dark:border-slate-800 dark:bg-slate-900/50">
                                            <tr>
                                                <th className="px-6 py-3 font-semibold text-slate-900 dark:text-white">Event Type</th>
                                                <th className="px-6 py-3 font-semibold text-slate-900 dark:text-white">Object</th>
                                                <th className="px-6 py-3 font-semibold text-slate-900 dark:text-white">Created At</th>
                                                <th className="px-6 py-3 font-semibold text-slate-900 dark:text-white text-right">ID</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                                            {events.map((evt) => (
                                                <tr key={evt.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/50">
                                                    <td className="px-6 py-4">
                                                        <code className="rounded bg-slate-100 px-1.5 py-0.5 text-xs font-semibold text-slate-900 dark:bg-slate-800 dark:text-white">
                                                            {evt.type}
                                                        </code>
                                                    </td>
                                                    <td className="px-6 py-4 font-mono text-xs text-slate-500">
                                                        {evt.object_type}:{evt.object_id?.substring(0, 8)}...
                                                    </td>
                                                    <td className="px-6 py-4 text-slate-500">
                                                        {new Date(evt.created_at).toLocaleString()}
                                                    </td>
                                                    <td className="px-6 py-4 text-right font-mono text-xs text-slate-400">
                                                        {evt.id?.substring(0, 12)}...
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            )}
                        </div>
                    </section>
                )}
            </div>

            {/* Success Modal for Key Generation */}
            <Modal isOpen={isModalOpen} onClose={() => setIsModalOpen(false)} title="New API Key Generated">
                <div className="flex flex-col gap-4">
                    <div className="flex items-center gap-x-2 rounded-lg bg-amber-50 dark:bg-amber-900/30 p-4 text-amber-800 dark:text-amber-300 ring-1 ring-inset ring-amber-200 dark:ring-amber-900/50">
                        <span className="material-symbols-outlined flex-shrink-0">warning</span>
                        <p className="text-sm font-medium">Copy your secret API key and store it securely. You will not be able to see it again.</p>
                    </div>
                    <div className="relative">
                        <input
                            type="text"
                            readOnly
                            value={generatedKey || ''}
                            className="form-input block w-full rounded-lg border-slate-300 dark:border-slate-700 bg-slate-50 dark:bg-slate-950 pr-12 text-slate-900 dark:text-white font-mono text-sm h-11"
                        />
                        <button
                            onClick={() => navigator.clipboard.writeText(generatedKey)}
                            className="absolute inset-y-0 right-0 flex items-center justify-center rounded-r-lg px-3 text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                            title="Copy to clipboard"
                        >
                            <span className="material-symbols-outlined text-lg">content_copy</span>
                        </button>
                    </div>
                    <div className="flex justify-end pt-4">
                        <button
                            onClick={() => setIsModalOpen(false)}
                            className="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-primary/90"
                        >
                            Done
                        </button>
                    </div>
                </div>
            </Modal>

            {/* Add Webhook Modal */}
            <Modal isOpen={isWebhookModalOpen} onClose={() => { setIsWebhookModalOpen(false); setCreatedWebhookSecret(null); }} title={createdWebhookSecret ? "Webhook Created" : "Add Webhook Endpoint"}>
                {createdWebhookSecret ? (
                    <div className="flex flex-col gap-4">
                        <div className="flex items-center gap-x-2 rounded-lg bg-green-50 dark:bg-green-900/30 p-4 text-green-800 dark:text-green-300 ring-1 ring-inset ring-green-200 dark:ring-green-900/50">
                            <span className="material-symbols-outlined flex-shrink-0">check_circle</span>
                            <p className="text-sm font-medium">Webhook endpoint created successfully!</p>
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Signing Secret</label>
                            <p className="text-xs text-slate-500 mb-2">Store this securely. You won't be able to see it again.</p>
                            <div className="relative">
                                <input
                                    type="text"
                                    readOnly
                                    value={createdWebhookSecret}
                                    className="form-input block w-full rounded-lg border-slate-300 dark:border-slate-700 bg-slate-50 dark:bg-slate-950 pr-12 text-slate-900 dark:text-white font-mono text-sm h-11"
                                />
                                <button
                                    onClick={() => navigator.clipboard.writeText(createdWebhookSecret)}
                                    className="absolute inset-y-0 right-0 flex items-center justify-center rounded-r-lg px-3 text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                                    title="Copy to clipboard"
                                >
                                    <span className="material-symbols-outlined text-lg">content_copy</span>
                                </button>
                            </div>
                        </div>
                        <div className="flex justify-end pt-4">
                            <button
                                onClick={() => { setIsWebhookModalOpen(false); setCreatedWebhookSecret(null); }}
                                className="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-primary/90"
                            >
                                Done
                            </button>
                        </div>
                    </div>
                ) : (
                    <div className="flex flex-col gap-4">
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Endpoint URL</label>
                            <input
                                type="url"
                                value={newWebhook.url}
                                onChange={(e) => setNewWebhook(prev => ({ ...prev, url: e.target.value }))}
                                placeholder="https://example.com/webhooks/recurso"
                                className="form-input block w-full rounded-lg border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white text-sm h-11"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Events to receive</label>
                            <div className="grid grid-cols-2 gap-2 max-h-48 overflow-y-auto p-2 border border-slate-200 dark:border-slate-700 rounded-lg">
                                {eventTypes.map(eventType => (
                                    <label key={eventType} className="flex items-center gap-2 text-sm cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800 p-2 rounded">
                                        <input
                                            type="checkbox"
                                            checked={newWebhook.events.includes(eventType)}
                                            onChange={() => toggleEventType(eventType)}
                                            className="rounded border-slate-300 text-primary focus:ring-primary"
                                        />
                                        <code className="text-xs text-slate-600 dark:text-slate-400">{eventType}</code>
                                    </label>
                                ))}
                            </div>
                        </div>
                        <div className="flex justify-end gap-3 pt-4">
                            <button
                                onClick={() => setIsWebhookModalOpen(false)}
                                className="rounded-lg px-4 py-2 text-sm font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleCreateWebhook}
                                className="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-primary/90"
                            >
                                Create Endpoint
                            </button>
                        </div>
                    </div>
                )}
            </Modal>
        </div>
    )
}

export default Developers
