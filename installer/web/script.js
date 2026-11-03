// Переход к следующему шагу
function nextStep(currentId, nextId) {
  document.getElementById(currentId).style.display = 'none';
  document.getElementById(nextId).style.display = 'block';
}

// Возврат к предыдущему шагу
function prevStep(currentId, prevId) {
  document.getElementById(currentId).style.display = 'none';
  document.getElementById(prevId).style.display = 'block';
}

// Проверка SSH (имитация)
function testSsh() {
  const statusEl = document.getElementById('sshStatus');
  statusEl.className = 'success';
  statusEl.textContent = 'Подключение успешно (имитация).';
  // В реальной версии тут будет fetch к бэкенду, который проверит SSH
}

// Имитация получения списка дисков
function fetchDisks() {
  const disksList = document.getElementById('disksList');
  disksList.innerHTML = '';

  // Имитируем ответ от сервера (в реальности — fetch к API)
  const mockDisks = [
    { path: '/dev/sda1', size: '100 GB', fs: 'ext4' },
    { path: '/dev/sdb1', size: '500 GB', fs: 'xfs' },
    { path: '/dev/nvme0n1p1', size: '1 TB', fs: 'btrfs' }
  ];

  mockDisks.forEach(disk => {
    const div = document.createElement('div');
    div.className = 'disk-item';
    div.innerHTML = `
      <input type="checkbox" id="disk_${disk.path}" value="${disk.path}">
      <label for="disk_${disk.path}">
        ${disk.path} — ${disk.size} (${disk.fs})
      </label>
    `;
    disksList.appendChild(div);
  });
}

// Формирование итоговой конфигурации
function generateConfig() {
  // Собираем данные
  const hosts = document.getElementById('hosts').value.trim().split('\n').filter(h => h);
  const sshUser = document.getElementById('sshUser').value;
  const sshPort = parseInt(document.getElementById('sshPort').value);
  const sshKeyPath = document.getElementById('sshKeyPath').value;
  const usePassword = document.getElementById('usePassword').checked;
  const sshPassword = usePassword ? document.getElementById('sshPassword').value : null;

  const selectedDisks = [];
  document.querySelectorAll('#disksList input[type="checkbox"]:checked').forEach(cb => {
    selectedDisks.push(cb.value);
  });

  const ydbSource = document.querySelector('input[name="ydbSource"]:checked').id;
  let ydbVersion = null, binPath = null;
  if (ydbSource === 'versionOnline') {
    ydbVersion = document.getElementById('ydbVersion').value;
  } else {
    binPath = document.getElementById('binPath').value;
  }

  const clusterScheme = document.querySelector('input[name="clusterScheme"]:checked').value;

  // Формируем объект конфигурации
  const config = {
    hosts: hosts,
    ssh: {
      user: sshUser,
      port: sshPort,
      keyPath: sshKeyPath,
      usePassword: usePassword,
      password: sshPassword
    },
    disks: selectedDisks,
    ydb: {
      version: ydbVersion,
      binPath: binPath
    },
    clusterScheme: clusterScheme,
    generatedAt: new Date().toISOString()
  };

  // Выводим в интерфейс
  document.getElementById('configOutput').textContent = JSON.stringify(config, null, 2);
  document.getElementById('result').style.display = 'block';
  document.getElementById('step-scheme').style.display = 'none';
}

// Скачивание конфигурации как файла
function downloadConfig() {
  const configText = document.getElementById('configOutput').textContent;
  const blob = new Blob([configText], { type: 'application/json' });
  const url = URL.createObjectURL(blob);

  const a = document.createElement('a');
  a.style.display = 'none';
  a.href = url;
  a.download = 'ydb-config.json';
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(url);
  document.body.removeChild(a);
}

// Сброс формы
function resetForm() {
  document.getElementById('result').style.display = 'none';
  document.getElementById('step-hosts').style.display = 'block';
  document.getElementById('hosts').value = '';
  document.getElementById('sshUser').value = 'root';
  document.getElementById('sshPort').value = 22;
  document.getElementById('sshKeyPath').value = '';
  document.getElementById('usePassword').checked = false;
  document.getElementById('sshPassword').value = '';
  document.getElementById('disksList').innerHTML = '';
  document.getElementById('ydbVersion').value = '23.1.34';
  document.getElementById('binPath').value = '';
  document.querySelector('#versionOnline').checked = true;
  document.querySelector('#schemeMirror').checked = true;
}

// Инициализация: показать первый шаг
window.onload = () => {
  document.getElementById('step-ssh').style.display = 'none';
  document.getElementById('step-disks').style.display = 'none';
  document.getElementById('step-version').style.display = 'none';
  document.getElementById('step-scheme').style.display = 'none';
  document.getElementById('result').style.display = 'none';

  // Показать поле пароля, если отмечена галочка
  document.getElementById('usePassword').onchange = function() {
    document.getElementById('passwordField').style.display = this.checked ? 'block' : 'none';
  };
};
