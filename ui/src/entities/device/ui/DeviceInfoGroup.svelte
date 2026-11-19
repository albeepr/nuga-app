<script lang="ts">
  import { FormGroup, FormRow, Typography } from '@naco-ui/svelte'
  import { fsd } from 'feature-sliced-svelte'

  import type { KeyboardStatus } from '../model/types'

  export let name: string
  export let path: string
  export let firmware: string
  export let status: KeyboardStatus | null = null
</script>

<div use:fsd={'entities/DeviceInfoGroup'}>
  <FormGroup>
    <FormRow title="Name">
      <Typography>{name}</Typography>
    </FormRow>
    <FormRow title="Path">
      <Typography>{path}</Typography>
    </FormRow>
    <FormRow title="Firmware version">
      <Typography>{firmware}</Typography>
    </FormRow>
    {#if status}
      <FormRow title="Connection">
        <Typography>{status.connection}</Typography>
      </FormRow>
      <FormRow title="Battery">
        <Typography
          >{status.battery}%{status.charging ? ' (charging)' : ''}</Typography
        >
      </FormRow>
      <FormRow title="Mac/Win switch">
        <Typography>{status.os}</Typography>
      </FormRow>
    {/if}
  </FormGroup>
</div>
