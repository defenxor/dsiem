import React from 'react'
import ReactJson from 'react-json-view'

const baseUrl = 'http://' + window.location.hostname
const dirFile = 'directives_demo.json'
// this one goes through nginx reverse proxy to avoid CORS
const appPort = window.location.port
const jsonUrl = baseUrl + ':' + appPort + '/dsiem/config/' + dirFile

export class JsonViewer extends React.Component {
  constructor () {
    super()
    this.state = {
      directives: {},
      status: 'Loading ..'
    }
  }

  async readJson () {
    try {
      const response = await fetch(jsonUrl)
      if (response.status === 200) {
        const j = await response.json()
        console.log('response.Json: ', j)
        console.log('response.text: ', response.text)
        this.setState({ directives: j, status: 'success' })
      } else {
        this.setState({
          status:
            'Failed to load ' +
            dirFile +
            '. HTTP status code: ' +
            response.status
        })
      }
    } catch (err) {
      this.setState({
        status: 'Error loading ' + dirFile + '. Error message: ' + err
      })
    }
  }

  componentDidMount () {
    this.readJson()
  }

  render () {
    if (this.state.status === 'success') {
      return <ReactJson src={this.state.directives} displayDataTypes={false} />
    } else {
      return this.state.status
    }
  }
}
