import React from 'react'

const Icon = ({ name, className = "", filled = false }) => {
    return (
        <span className={`material-symbols-outlined ${filled ? 'fill' : ''} ${className}`}>
            {name}
        </span>
    )
}

export default Icon
