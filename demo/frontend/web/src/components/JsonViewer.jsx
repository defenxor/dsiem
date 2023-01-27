import React, { useState, useEffect } from 'react'
import ReactJson from 'react-json-view'
import { fetchUrl } from './utils.js'
import { PropTypes } from 'prop-types'

export const JsonViewer = () => {
  const [result, setResult] = useState({})
  const [status, setStatus] = useState('Loading ..')

  const directiveFile = 'directives_demo.json'
  const configUrl = `${window.location.protocol}//${window.location.hostname}:${
      window.location.port
    }/dsiem/config`
  const targetUrl = `${configUrl}/${directiveFile}`

  useEffect(
    () => {
      fetchUrl(targetUrl).then(val => {
        setResult(val.result)
        setStatus(val.status)
      })
    },
    [targetUrl]
  )

  if (status === 'success') {
    return <ReactJson src={result} displayDataTypes={false} />
  } else {
    return status
  }
}

JsonViewer.propTypes = {
  configUrl: PropTypes.string,
  match: PropTypes.shape({
    params: PropTypes.shape({
      directiveFile: PropTypes.string
    })
  })
}
