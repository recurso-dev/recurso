import React from 'react'

// Skeleton loader for cards
export const CardSkeleton = ({ className = "" }) => (
    <div className={`animate-pulse rounded-xl bg-slate-100 dark:bg-slate-800 p-6 ${className}`}>
        <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-1/3 mb-4" />
        <div className="h-8 bg-slate-200 dark:bg-slate-700 rounded w-1/2" />
    </div>
)

// Skeleton loader for table rows
export const TableRowSkeleton = ({ columns = 5 }) => (
    <tr className="animate-pulse">
        {[...Array(columns)].map((_, i) => (
            <td key={i} className="px-6 py-4">
                <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-3/4" />
            </td>
        ))}
    </tr>
)

// Skeleton loader for list items
export const ListSkeleton = ({ rows = 5 }) => (
    <div className="space-y-4">
        {[...Array(rows)].map((_, i) => (
            <div key={i} className="animate-pulse flex items-center gap-4 p-4 bg-slate-100 dark:bg-slate-800 rounded-lg">
                <div className="w-10 h-10 bg-slate-200 dark:bg-slate-700 rounded-full" />
                <div className="flex-1">
                    <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-1/3 mb-2" />
                    <div className="h-3 bg-slate-200 dark:bg-slate-700 rounded w-1/2" />
                </div>
            </div>
        ))}
    </div>
)

// Chart skeleton
export const ChartSkeleton = ({ className = "" }) => (
    <div className={`animate-pulse bg-slate-100 dark:bg-slate-800 rounded-xl p-6 ${className}`}>
        <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-1/4 mb-6" />
        <div className="flex items-end justify-between gap-2 h-40">
            {[40, 65, 45, 80, 55, 70, 85, 60, 75, 50, 90, 65].map((height, i) => (
                <div
                    key={i}
                    className="flex-1 bg-slate-200 dark:bg-slate-700 rounded-t"
                    style={{ height: `${height}%` }}
                />
            ))}
        </div>
    </div>
)

// Stats card skeleton
export const StatsSkeleton = () => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {[...Array(4)].map((_, i) => (
            <CardSkeleton key={i} />
        ))}
    </div>
)

export default {
    CardSkeleton,
    TableRowSkeleton,
    ListSkeleton,
    ChartSkeleton,
    StatsSkeleton
}
