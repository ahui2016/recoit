function ajaxPost(form, url, btn, onloadHandler) {
  if (btn) {
    btn.prop('disabled', true);
  }
  let xhr = new XMLHttpRequest();

  xhr.responseType = 'json';
  xhr.open('POST', url);

  xhr.onerror = function () {
    window.alert('An error occurred during the transaction');
  };
  
  xhr.onload = onloadHandler;

  xhr.addEventListener('loadend', function() {
    if (btn) {
      btn.prop('disabled', false);
    }
  });

  xhr.send(form);
}

function ajaxGet(url, btn, onloadHandler) {
  if (btn) {
    btn.prop('disabled', true);
  }
  let xhr = new XMLHttpRequest();

  xhr.responseType = 'json';
  xhr.open('GET', url);

  xhr.onerror = function () {
    window.alert('An error occurred during the transaction');
  }
  
  xhr.onload = onloadHandler;

  xhr.addEventListener('loadend', function() {
    if (btn) {
      btn.prop('disabled', false);
    }
  });

  xhr.send();
}

function insertErrorAlert(errMsg) {
  console.log(errMsg);
  let errAlert = $('#alert-danger-tmpl').contents().clone();
  errAlert.find('.AlertMessage').text(errMsg);
  errAlert.insertAfter('#alert-danger-tmpl');
}

function insertSuccessAlert(errMsg) {
  console.log(errMsg);
  let errAlert = $('#alert-success-tmpl').contents().clone();
  errAlert.find('.AlertMessage').text(errMsg);
  errAlert.insertAfter('#alert-success-tmpl');
}

function fileSizeToString(fileSize) {
  sizeMB = fileSize / 1024 / 1024;
  if (sizeMB < 1) {
      return `${(sizeMB * 1024).toFixed(2)} KB`;
  }
  return `${sizeMB.toFixed(2)} MB`;
}

function getNewTags() {
  let trimmed = $('#tags-input').val().replace(/#|,|，/g, ' ').trim();
  if (trimmed.length == 0) {
    return [];
  }
  return trimmed.split(/ +/);
}

function addPrefix(arr, prefix) {
  return arr.map(x => prefix + x).join(' ');
}

function checkHashHex(hashHex) {
  let form = new FormData();
  form.append('hashHex', hashHex);
  ajaxPost(form, '/api/checksum', null, function() {
    if (this.status == 200) {
      console.log('OK');
    } else {
      console.log(`Error: ${this.status} ${JSON.stringify(this.response)}`);
    }
  });
}

// https://developer.mozilla.org/en-US/docs/Web/API/SubtleCrypto/digest
async function sha256Hex(file) {
  let buffer = await file.arrayBuffer();
  const hashBuffer = await crypto.subtle.digest('SHA-256', buffer);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
  return hashHex;
}

function simpleDateTime(date) {
  return date.toString().split(' ').slice(1, 5).join(' ')
}

function simpleDate(date) {
  let year = '' + date.getFullYear(),
      month = '' + (date.getMonth() + 1),
      day = '' + date.getDate();
  if (month.length < 2) month = '0' + month;
  if (day.length < 2) day = '0' + day;
  return [year, month, day].join('-');
}
