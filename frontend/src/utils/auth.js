/**
 * Copyright 2023 The CubeFS Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
 * implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

function checkValue(arr, key, value) {
  for (let i = 0; i < arr.length; i++) {
    if (arr[i][key] === value) {
      return true
    }
  }
  return false
}

export function authMixin (Vue, { router, store }) {
  router.beforeEach((to, from, next) => {
    if (to.name === 'loginPage') {
      next()
    } else if (!localStorage.getItem('userInfo')) {
      next({ name: 'loginPage' })
    } else {
      const { allAuth } = store.state.moduleUser
      if (!allAuth) {
        store.dispatch('moduleUser/setAuth').then(() => {
          next()
        }).catch(() => {
          next({ name: 'loginPage' })
        })
      } else {
        next()
      }
    }
  })
  Vue.directive('auth', {
    inserted: (el, binding) => {
      const { allAuth } = store.state.moduleUser
      const frontEndAuth = allAuth?.filter(item => item.is_check)
      if (!checkValue(frontEndAuth, 'auth_code', binding.value)) {
        el.parentNode.removeChild(el)
      }
    },
  })
}

export function getCodeList(that) {
  return [
    {
      title: that.$t('auth.cluster'),
      children: ['CLUSTER_CREATE', 'CLUSTER_UPDATE', 'CLUSTER_DELETE', 'CFS_USERS_LIST', 'CFS_USERS_CREATE', 'CFS_USERS_DELETE', 'CFS_USERS_VOLS_TRANSFER'],
    },
    {
      title: that.$t('auth.volume'),
      children: ['CFS_VOLS_CREATE', 'CFS_VOLS_UPDATE', 'CFS_VOLS_EXPAND', 'CFS_VOLS_SHRINK', 'CFS_USERS_POLICIES', 'CFS_USERS_POLICIES_DELETE'],
    },
    {
      title: that.$t('auth.multi_replicas'),
      children: ['CFS_DATANODE_DECOMMISSION', 'CFS_DATANODE_MIGRATE', 'CFS_DISKS_DECOMMISSION', 'CFS_DATAPARTITION_DECOMMISSION'],
    },
    {
      title: that.$t('auth.ec'),
      children: ['BLOBSTORE_NODES_ACCESS', 'BLOBSTORE_DISKS_ACCESS', 'BLOBSTORE_DISKS_SET', 'BLOBSTORE_DISKS_PROBE'],
    },
    {
      title: that.$t('auth.meta'),
      children: ['CFS_METANODE_DECOMMISSION', 'CFS_METANODE_MIGRATE', 'CFS_METAPARTITION_DECOMMISSION'],
    },
    {
      title: that.$t('auth.sync'),
      children: [
        'CFS_SYNCNODE_LIST', 'CFS_SYNCNODE_TASKS', 'CFS_SYNCNODE_VERSION', 'CFS_SYNCNODE_STAT', 'CFS_SYNCNODE_DISPATCH', 'CFS_SYNCNODE_RELOAD', 'CFS_SYNCNODE_DECOMMISSION', 'CFS_SYNCNODE_DRAIN', 'CFS_SYNCNODE_RESTORE',
        'CFS_SYNCRULE_LIST', 'CFS_SYNCRULE_GET', 'CFS_SYNCRULE_CREATE', 'CFS_SYNCRULE_UPDATE', 'CFS_SYNCRULE_DELETE', 'CFS_SYNCRULE_PAUSE', 'CFS_SYNCRULE_RESUME', 'CFS_SYNCRULE_TRIGGER',
        'CFS_SYNCTASK_LIST', 'CFS_SYNCTASK_GET', 'CFS_SYNCTASK_CANCEL', 'CFS_SYNCTASK_RETRY', 'CFS_SYNCTASK_EXPORT',
        'CFS_SYNCBACKEND_LIST', 'CFS_SYNCBACKEND_CREATE', 'CFS_SYNCBACKEND_UPDATE', 'CFS_SYNCBACKEND_DELETE', 'CFS_SYNCBACKEND_CONFIG',
      ],
    },
    {
      title: that.$t('auth.bench'),
      children: [
        'CFS_BENCHRULE_LIST', 'CFS_BENCHRULE_GET', 'CFS_BENCHRULE_CREATE', 'CFS_BENCHRULE_UPDATE', 'CFS_BENCHRULE_DELETE', 'CFS_BENCHRULE_TRIGGER',
        'CFS_BENCHTASK_LIST', 'CFS_BENCHTASK_GET', 'CFS_BENCHTASK_CANCEL', 'CFS_BENCHTASK_RETRY',
      ],
    },
    {
      title: that.$t('auth.posixcheck'),
      children: [
        'CFS_POSIXCHECK_LIST', 'CFS_POSIXCHECK_GET', 'CFS_POSIXCHECK_RUN', 'CFS_POSIXCHECK_CANCEL',
      ],
    },
    {
      title: that.$t('auth.file'),
      children: ['CFS_S3_DIRS_CREATE', 'CFS_S3_FILES_DOWNLOAD_SIGNEDURL', 'CFS_S3_FILES_UPLOAD_SIGNEDURL'],
    },
    {
      title: that.$t('auth.clusterevents'),
      children: ['BLOBSTORE_CONFIG_SET'],
    },
    {
      title: that.$t('auth.user'),
      children: ['AUTH_USER_CREATE', 'AUTH_USER_UPDATE', 'AUTH_USER_DELETE', 'AUTH_USER_PASSWORD_UPDATE'],
    },
    {
      title: that.$t('auth.role'),
      children: ['AUTH_ROLE_CREATE', 'AUTH_ROLE_UPDATE', 'AUTH_ROLE_DELETE'],
    },
  ]
}

export const backendAuthids = [3, 4, 7, 11, 13, 14, 15, 16, 17, 18, 19, 24, 25, 26, 31, 32, 36, 41, 42, 43, 44, 48, 49, 53, 54, 58, 59, 63, 64, 66, 67, 70, 71]
