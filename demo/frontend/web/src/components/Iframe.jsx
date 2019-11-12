import React from 'react'
import PropTypes from 'prop-types'

export const Iframe = props => (
  <iframe
    src={props.src}
    height={props.height}
    width={props.width}
    title={props.title}
  />
)

Iframe.propTypes = {
  src: PropTypes.string,
  title: PropTypes.string,
  height: PropTypes.string,
  width: PropTypes.string
}
