import React from 'react'
import { endpoints } from '../lib/api'

const Notifications = () => {
    const [notifications, setNotifications] = React.useState([])
    const [loading, setLoading] = React.useState(true)

    React.useEffect(() => {
        const fetchNotifications = async () => {
            try {
                const response = await endpoints.getEvents({ limit: 20 })
                // Map events to notification format
                const mapped = (response.data.data || []).map(evt => {
                    let title = "System Event"
                    let description = `Event ${evt.type} occurred on ${evt.object_type}`
                    let icon = "info"

                    switch (evt.type) {
                        case 'subscription.created':
                            title = "New Subscription"
                            description = "A new subscription was created."
                            icon = "group_add"
                            break
                        case 'invoice.paid':
                            title = "Payment Received"
                            description = "Invoice was successfully paid."
                            icon = "payments"
                            break
                        case 'invoice.payment_failed':
                            title = "Payment Failed"
                            description = "Payment processing failed for invoice."
                            icon = "error"
                            break
                        case 'customer.created':
                            title = "New Customer"
                            description = "A new customer has been registered."
                            icon = "person_add"
                            break
                        default:
                            title = evt.type.replace('.', ' ').replace(/\b\w/g, l => l.toUpperCase())
                            icon = "circle_notifications"
                    }

                    return {
                        id: evt.id,
                        title,
                        description,
                        time: new Date(evt.created_at).toLocaleString(), // Simple formatting
                        icon,
                        read: false // No backend support yet
                    }
                })
                setNotifications(mapped)
            } catch (error) {
                console.error("Failed to fetch notifications:", error)
            } finally {
                setLoading(false)
            }
        }
        fetchNotifications()
    }, [])

    return (
        <div className="flex flex-col max-w-3xl mx-auto">
            <header className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">Notifications</h1>
                    <p className="mt-1 text-sm text-gray-500 dark:text-zinc-400">Stay updated with the latest events.</p>
                </div>
                <button className="text-sm font-medium text-black hover:text-gray-700 dark:text-white dark:hover:text-gray-300">
                    Mark all as read
                </button>
            </header>

            <div className="flex flex-col gap-4">
                {loading ? (
                    <div className="text-center py-10 text-gray-500">Loading notifications...</div>
                ) : notifications.length === 0 ? (
                    <div className="text-center py-10 text-gray-500 bg-gray-50 dark:bg-zinc-900 rounded-xl border border-dashed border-gray-200 dark:border-zinc-800">
                        No notifications found.
                    </div>
                ) : (
                    notifications.map((note) => (
                        <div
                            key={note.id}
                            className={`flex items-start gap-4 rounded-xl border p-4 transition-all bg-white border-gray-200 shadow-sm dark:bg-zinc-900 dark:border-zinc-700`}
                        >
                            <div className={`flex items-center justify-center size-10 rounded-lg shrink-0 bg-black text-white dark:bg-white dark:text-black`}>
                                <span className="material-symbols-outlined text-[20px]">{note.icon}</span>
                            </div>
                            <div className="flex-1">
                                <div className="flex items-center justify-between">
                                    <h3 className={`text-sm font-semibold text-gray-900 dark:text-white`}>
                                        {note.title}
                                    </h3>
                                    <span className="text-xs text-gray-400">{note.time}</span>
                                </div>
                                <p className="mt-1 text-sm text-gray-500 dark:text-zinc-400">
                                    {note.description}
                                </p>
                            </div>
                        </div>
                    ))
                )}
            </div>
        </div>
    )
}

export default Notifications
