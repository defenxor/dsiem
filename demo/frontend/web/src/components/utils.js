export const sleep = ms => new Promise(resolve => setTimeout(resolve, ms))

export const isChrome = () => {
  try {
    const isChromium = window.chrome
    const winNav = window.navigator
    const vendorName = winNav.vendor
    const isOpera = typeof window.opr !== 'undefined'
    const isIEedge = winNav.userAgent.indexOf('Edge') > -1
    const isIOSChrome = winNav.userAgent.match('CriOS')
    if (isIOSChrome) {
      // is Google Chrome on IOS
      return false
    }

    if (
      isChromium !== null &&
      typeof isChromium !== 'undefined' &&
      vendorName === 'Google Inc.' &&
      isOpera === false &&
      isIEedge === false
    ) {
      return true
    }
  } catch (e) {
    return false
  }
  return false
}
