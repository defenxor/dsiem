import React from 'react'
import { HashRouter as Router, Switch, Route } from 'react-router-dom'
import { DemoMenu } from './demomenu.jsx'
import { DemoOverview } from './demooverview.jsx'
import { JsonViewer } from './jsonviewer.jsx'
import { EuiPage } from '@elastic/eui'

export const App = props => (
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
