import React from 'react'
import ReactDOM from 'react-dom'
import { App } from './components/app.jsx'
import { BrowserRouter } from 'react-router-dom'
import './styles.css'

ReactDOM.render(
  <BrowserRouter>
    <App />
  </BrowserRouter>,
  document.getElementById('root')
)
