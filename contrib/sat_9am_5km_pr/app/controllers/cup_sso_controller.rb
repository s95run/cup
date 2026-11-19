# frozen_string_literal: true

class CupSsoController < ApplicationController
  before_action :authenticate_user!

  def show
    athlete = current_user.athlete
    return redirect_to root_path, alert: 'Профиль атлета не найден' unless athlete

    jti = SecureRandom.uuid
    payload = {
    user_id: current_user.id,
    athlete_id: athlete.id,
    athlete_code: athlete.code,
    name: athlete.name,
    gender: athlete.gender,
    home_event_code: athlete.event&.code_name,
    is_admin: current_user.admin?,
    jti: jti,
    state: params[:state].to_s,
    exp: 60.seconds.from_now.to_i
  }
    token = Cup::SsoToken.build(payload)
    return_to = sanitize_return_to(params[:return_to])
    redirect_to "#{return_to}?token=#{ERB::Util.url_encode(token)}&state=#{ERB::Util.url_encode(params[:state].to_s)}",
                allow_other_host: true
  end

  private

  def sanitize_return_to(url)
    uri = URI.parse(url.to_s)
    allowed = ENV.fetch('CUP_CALLBACK_HOSTS', 'cup.s95.ru,localhost').split(',')
    host = uri.host.to_s
    raise ActionController::BadRequest unless allowed.any? { |h| host == h.strip || host.end_with?(".#{h.strip}") }

    url.to_s
  rescue URI::InvalidURIError
    raise ActionController::BadRequest
  end
end