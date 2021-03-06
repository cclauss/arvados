# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::ContainersController < ApplicationController
  accept_attribute_as_json :environment, Hash
  accept_attribute_as_json :mounts, Hash
  accept_attribute_as_json :runtime_constraints, Hash
  accept_attribute_as_json :command, Array
  accept_attribute_as_json :scheduling_parameters, Hash

  skip_before_filter :find_object_by_uuid, only: [:current]
  skip_before_filter :render_404_if_no_object, only: [:current]

  def auth
    if @object.locked_by_uuid != Thread.current[:api_client_authorization].uuid
      raise ArvadosModel::PermissionDeniedError.new("Not locked by your token")
    end
    @object = @object.auth
    show
  end

  # Updates use row locking to resolve races between multiple
  # dispatchers trying to lock the same container.
  def update
    @object.with_lock do
      super
    end
  end

  def find_objects_for_index
    super
    if action_name == 'lock' || action_name == 'unlock'
      # Avoid loading more fields than we need
      @objects = @objects.select(:id, :uuid, :state, :priority, :auth_uuid, :locked_by_uuid)
      @select = %w(uuid state priority auth_uuid locked_by_uuid)
    end
  end

  def lock
    @object.lock
    show
  end

  def unlock
    @object.unlock
    show
  end

  def current
    if Thread.current[:api_client_authorization].nil?
      send_error("Not logged in", status: 401)
    else
      c = Container.where(auth_uuid: Thread.current[:api_client_authorization].uuid).first
      if c.nil?
        send_error("Token is not associated with a container.", status: 404)
      else
        @object = c
        show
      end
    end
  end
end
