import React from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  componentDidCatch(error, errorInfo) {
    this.setState({ errorInfo });
    console.error('[ErrorBoundary]', error, errorInfo);
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
  };

  render() {
    if (this.state.hasError) {
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
          <AlertTriangle size={48} color="var(--color-warning, #f59e0b)" />
          <h2 style={{ marginTop: '1rem', fontSize: '1.25rem', fontWeight: 600, color: 'var(--color-text-primary, #fff)' }}>
            Something went wrong
          </h2>
          <p style={{ marginTop: '0.5rem', color: 'var(--color-text-secondary, #94a3b8)', maxWidth: '400px' }}>
            {this.state.error?.message || 'An unexpected error occurred. Please try again.'}
          </p>
          <button
            onClick={this.handleRetry}
            style={{
              marginTop: '1.5rem',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
              padding: '0.5rem 1rem',
              borderRadius: '0.5rem',
              border: '1px solid var(--color-border, rgba(255,255,255,0.1))',
              background: 'var(--color-surface, rgba(255,255,255,0.05))',
              color: 'var(--color-text-primary, #fff)',
              cursor: 'pointer',
              fontSize: '0.875rem',
              fontWeight: 500,
            }}
          >
            <RefreshCw size={16} /> Try Again
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
