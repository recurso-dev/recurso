
/**
 * Loading skeleton component that mimics content shape while loading.
 * Variants: text, card, table, stat
 */
export function Skeleton({ variant = 'text', count = 1, style = {} }) {
    const baseStyle = {
        background: 'linear-gradient(90deg, rgba(255,255,255,0.04) 25%, rgba(255,255,255,0.08) 50%, rgba(255,255,255,0.04) 75%)',
        backgroundSize: '200% 100%',
        animation: 'shimmer 1.5s ease-in-out infinite',
        borderRadius: '0.5rem',
        ...style,
    };

    const variants = {
        text: { height: '1rem', marginBottom: '0.5rem' },
        card: { height: '8rem', marginBottom: '1rem' },
        stat: { height: '5rem', marginBottom: '0.75rem' },
        table: { height: '3rem', marginBottom: '0.25rem' },
        avatar: { height: '2.5rem', width: '2.5rem', borderRadius: '50%' },
    };

    return (
        <>
            <style>{`
        @keyframes shimmer {
          0% { background-position: -200% 0; }
          100% { background-position: 200% 0; }
        }
      `}</style>
            {Array.from({ length: count }).map((_, i) => (
                <div key={i} style={{ ...baseStyle, ...variants[variant] }} />
            ))}
        </>
    );
}

/**
 * Page loading state with multiple skeleton rows
 */
export function PageSkeleton() {
    return (
        <div style={{ padding: '1.5rem' }}>
            <Skeleton variant="text" style={{ width: '200px', height: '1.5rem', marginBottom: '1.5rem' }} />
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '1rem', marginBottom: '2rem' }}>
                <Skeleton variant="stat" />
                <Skeleton variant="stat" />
                <Skeleton variant="stat" />
                <Skeleton variant="stat" />
            </div>
            <Skeleton variant="table" count={5} />
        </div>
    );
}

/**
 * Table loading state
 */
export function TableSkeleton({ rows = 5 }) {
    return (
        <div>
            <Skeleton variant="text" style={{ width: '150px', height: '1.25rem', marginBottom: '1rem' }} />
            <Skeleton variant="table" count={rows} />
        </div>
    );
}

/**
 * Empty state component
 */
export function EmptyState({ icon: Icon, title, description, action }) {
    return (
        <div style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '3rem',
            textAlign: 'center',
            minHeight: '200px'
        }}>
            {Icon && <Icon size={48} style={{ color: 'var(--color-text-tertiary, #475569)', marginBottom: '1rem' }} />}
            <h3 style={{ fontSize: '1.125rem', fontWeight: 600, color: 'var(--color-text-primary, #fff)' }}>
                {title}
            </h3>
            {description && (
                <p style={{ marginTop: '0.5rem', color: 'var(--color-text-secondary, #94a3b8)', maxWidth: '400px' }}>
                    {description}
                </p>
            )}
            {action && (
                <div style={{ marginTop: '1.5rem' }}>{action}</div>
            )}
        </div>
    );
}

/**
 * Error state component 
 */
export function ErrorState({ message, onRetry }) {
    return (
        <div style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '2rem',
            textAlign: 'center',
            border: '1px solid rgba(239, 68, 68, 0.2)',
            borderRadius: '0.75rem',
            background: 'rgba(239, 68, 68, 0.05)',
        }}>
            <p style={{ color: '#ef4444', fontWeight: 500 }}>
                {message || 'Failed to load data'}
            </p>
            {onRetry && (
                <button
                    onClick={onRetry}
                    style={{
                        marginTop: '1rem',
                        padding: '0.5rem 1rem',
                        borderRadius: '0.5rem',
                        border: '1px solid rgba(239, 68, 68, 0.3)',
                        background: 'transparent',
                        color: '#ef4444',
                        cursor: 'pointer',
                        fontSize: '0.875rem',
                    }}
                >
                    Retry
                </button>
            )}
        </div>
    );
}

export default Skeleton;
