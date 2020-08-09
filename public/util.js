function ajaxPost(form, url, btn, onloadHandler) {
  if (btn) {
    btn.prop('disabled', true);
  }
  let xhr = new XMLHttpRequest();

  xhr.responseType = 'json';
  xhr.open('POST', url);

  xhr.onerror = function () {
    window.alert('An error occurred during the transaction');
  }
  
  xhr.onload = onloadHandler;

  xhr.addEventListener('loadend', function() {
    if (btn) {
      btn.prop('disabled', false);
    }
  });

  xhr.send(form);
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
  return $('#tags-input').val().replace(/#|,|，/g, ' ').trim().split(/ +/);
}

function addPrefix(arr, prefix) {
  return arr.map(x => prefix + x).join(' ');
}