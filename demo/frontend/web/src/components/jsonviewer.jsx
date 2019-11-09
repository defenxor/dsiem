import React, { useState, useEffect } from 'react'
import ReactJson from 'react-json-view'
import { fetchUrl } from './utils.js'

export const JsonViewer = props => {
  const [result, setResult] = useState({})
  const [status, setStatus] = useState('Loading ..')

  const { directiveFile = 'directives_demo.json' } = props.match.params
  const {
    configUrl = `http://${window.location.hostname}:${
      window.location.port
    }/dsiem/config`
  } = props
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
