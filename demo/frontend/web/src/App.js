import React from 'react'
import { HashRouter as Router, Switch, Route } from 'react-router-dom'
import { DemoMenu } from './components/DemoMenu.jsx'
import { DemoOverview } from './components/DemoOverview.jsx'
import { JsonViewer } from './components/JsonViewer.jsx'
import { EuiPage } from '@elastic/eui'
import './App.css'

const App = props => (
  <Router>
    <EuiPage>
      <Switch>
        <Route exact path='/' component={DemoMenu} />
        <Route exact path='/directive/:directiveFile?' component={JsonViewer} />
        <Route exact path='/overview' component={DemoOverview} />
      </Switch>
    </EuiPage>
  </Router>
)

export default App
